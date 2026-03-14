package http_proxy_plugin

import (
	"hash/fnv"
	"math"
	"strconv"
	"strings"
	"sync"
)

const (
	defaultBloomFalsePositive = 0.01
	minBloomBits              = 256
	maxBloomHashes            = 12
)

type coreIPACL struct {
	signature string
	allowSet  map[string]struct{}
	denyBloom *bloomFilter
	denyRaw   []string
}

func (a *coreIPACL) IsAllowed(ip string) bool {
	if ip == "" {
		return true
	}
	if a != nil && a.allowSet != nil {
		if _, ok := a.allowSet[ip]; ok {
			return true
		}
	}
	if a != nil && a.denyBloom != nil && a.denyBloom.MightContain(ip) {
		return false
	}
	return true
}

type coreIPACLManager struct {
	mu        sync.RWMutex
	byService map[int64]*coreIPACL
}

var globalCoreIPACLManager = newCoreIPACLManager()

func GetCoreIPACLManager() *coreIPACLManager {
	return globalCoreIPACLManager
}

func newCoreIPACLManager() *coreIPACLManager {
	return &coreIPACLManager{
		byService: map[int64]*coreIPACL{},
	}
}

func (m *coreIPACLManager) GetOrBuild(serviceID int64, openAuth int, whiteList, blackList string) *coreIPACL {
	sig := buildCoreIPACLSignature(openAuth, whiteList, blackList)

	if serviceID > 0 {
		m.mu.RLock()
		if acl, ok := m.byService[serviceID]; ok && acl != nil && acl.signature == sig {
			m.mu.RUnlock()
			return acl
		}
		m.mu.RUnlock()
	}

	acl := buildCoreIPACL(sig, whiteList, blackList)
	if serviceID > 0 {
		m.mu.Lock()
		m.byService[serviceID] = acl
		m.mu.Unlock()
	}
	return acl
}

func (m *coreIPACLManager) ClearAllowList(serviceID int64) bool {
	if serviceID <= 0 {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	acl, ok := m.byService[serviceID]
	if !ok || acl == nil {
		return false
	}

	next := &coreIPACL{
		signature: acl.signature,
		allowSet:  map[string]struct{}{},
		denyBloom: acl.denyBloom,
		denyRaw:   append([]string{}, acl.denyRaw...),
	}
	m.byService[serviceID] = next
	return true
}

func (m *coreIPACLManager) ClearAllAllowLists() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for serviceID, acl := range m.byService {
		if acl == nil {
			continue
		}
		next := &coreIPACL{
			signature: acl.signature,
			allowSet:  map[string]struct{}{},
			denyBloom: acl.denyBloom,
			denyRaw:   append([]string{}, acl.denyRaw...),
		}
		m.byService[serviceID] = next
	}
}

func (m *coreIPACLManager) RebuildDenyBloom(serviceID int64) bool {
	if serviceID <= 0 {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	acl, ok := m.byService[serviceID]
	if !ok || acl == nil {
		return false
	}

	next := &coreIPACL{
		signature: acl.signature,
		allowSet:  acl.allowSet,
		denyBloom: newBloomFromList(acl.denyRaw),
		denyRaw:   append([]string{}, acl.denyRaw...),
	}
	m.byService[serviceID] = next
	return true
}

func (m *coreIPACLManager) RebuildAllDenyBloom() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for serviceID, acl := range m.byService {
		if acl == nil {
			continue
		}
		next := &coreIPACL{
			signature: acl.signature,
			allowSet:  acl.allowSet,
			denyBloom: newBloomFromList(acl.denyRaw),
			denyRaw:   append([]string{}, acl.denyRaw...),
		}
		m.byService[serviceID] = next
	}
}

func buildCoreIPACL(signature, whiteList, blackList string) *coreIPACL {
	denyRaw := parseIPList(blackList)
	return &coreIPACL{
		signature: signature,
		allowSet:  buildAllowSet(whiteList),
		denyBloom: newBloomFromList(denyRaw),
		denyRaw:   denyRaw,
	}
}

func buildCoreIPACLSignature(openAuth int, whiteList, blackList string) string {
	return strings.TrimSpace(strings.Join([]string{
		strconv.Itoa(openAuth),
		whiteList,
		blackList,
	}, "|"))
}

func parseIPList(list string) []string {
	list = strings.TrimSpace(list)
	if list == "" {
		return nil
	}
	parts := strings.Split(list, ",")
	out := make([]string, 0, len(parts))
	for _, raw := range parts {
		item := strings.TrimSpace(raw)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func buildAllowSet(list string) map[string]struct{} {
	parts := parseIPList(list)
	if len(parts) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(parts))
	for _, item := range parts {
		out[item] = struct{}{}
	}
	return out
}

type bloomFilter struct {
	m    uint64
	k    uint8
	bits []uint64
}

func newBloomFromList(values []string) *bloomFilter {
	if len(values) == 0 {
		return nil
	}
	bf := newBloomFilter(len(values), defaultBloomFalsePositive)
	if bf == nil {
		return nil
	}
	for _, v := range values {
		if v == "" {
			continue
		}
		bf.Add(v)
	}
	return bf
}

func newBloomFilter(n int, fpRate float64) *bloomFilter {
	if n <= 0 {
		return nil
	}
	if fpRate <= 0 || fpRate >= 1 {
		fpRate = defaultBloomFalsePositive
	}

	m := -float64(n) * math.Log(fpRate) / (math.Ln2 * math.Ln2)
	if m < float64(minBloomBits) {
		m = float64(minBloomBits)
	}
	mBits := uint64(math.Ceil(m))
	k := int(math.Round((float64(mBits) / float64(n)) * math.Ln2))
	if k < 1 {
		k = 1
	}
	if k > maxBloomHashes {
		k = maxBloomHashes
	}

	return &bloomFilter{
		m:    mBits,
		k:    uint8(k),
		bits: make([]uint64, (mBits+63)/64),
	}
}

func (b *bloomFilter) Add(value string) {
	if b == nil || b.m == 0 {
		return
	}
	h1, h2 := bloomHashes(value)
	for i := 0; i < int(b.k); i++ {
		bit := (h1 + uint64(i)*h2) % b.m
		b.bits[bit>>6] |= 1 << (bit & 63)
	}
}

func (b *bloomFilter) MightContain(value string) bool {
	if b == nil || b.m == 0 {
		return false
	}
	h1, h2 := bloomHashes(value)
	for i := 0; i < int(b.k); i++ {
		bit := (h1 + uint64(i)*h2) % b.m
		if (b.bits[bit>>6] & (1 << (bit & 63))) == 0 {
			return false
		}
	}
	return true
}

func bloomHashes(value string) (uint64, uint64) {
	h1 := fnv.New64a()
	_, _ = h1.Write([]byte(value))
	sum1 := h1.Sum64()

	h2 := fnv.New64()
	_, _ = h2.Write([]byte(value))
	sum2 := h2.Sum64()
	if sum2 == 0 {
		sum2 = 0x9e3779b97f4a7c15
	}

	return sum1, sum2
}
