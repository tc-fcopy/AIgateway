package security

import (
	"net"
	"strings"
	"sync"
)

// IPRestrictionManager checks global and consumer-level allow/deny rules.
type IPRestrictionManager struct {
	lock              sync.RWMutex
	enableCIDR        bool
	globalWhitelist   []string
	globalBlacklist   []string
	consumerWhitelist map[string][]string
	consumerBlacklist map[string][]string
}

func NewIPRestrictionManager() *IPRestrictionManager {
	return &IPRestrictionManager{
		enableCIDR:        true,
		consumerWhitelist: make(map[string][]string),
		consumerBlacklist: make(map[string][]string),
	}
}

func (m *IPRestrictionManager) SetGlobalRules(enableCIDR bool, whitelist, blacklist []string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.enableCIDR = enableCIDR
	m.globalWhitelist = append([]string{}, whitelist...)
	m.globalBlacklist = append([]string{}, blacklist...)
}

func (m *IPRestrictionManager) SetConsumerRules(consumer string, whitelist, blacklist []string) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if consumer == "" {
		return
	}
	m.consumerWhitelist[consumer] = append([]string{}, whitelist...)
	m.consumerBlacklist[consumer] = append([]string{}, blacklist...)
}

func (m *IPRestrictionManager) IsAllowed(ip, consumer string) bool {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if ip == "" {
		return false
	}

	// blacklist has priority
	if m.matchesAny(ip, m.globalBlacklist) || m.matchesAny(ip, m.consumerBlacklist[consumer]) {
		return false
	}

	whitelist := append([]string{}, m.globalWhitelist...)
	whitelist = append(whitelist, m.consumerWhitelist[consumer]...)
	if len(whitelist) == 0 {
		return true
	}

	return m.matchesAny(ip, whitelist)
}

func (m *IPRestrictionManager) matchesAny(ip string, rules []string) bool {
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "" {
			continue
		}

		if rule == "*" || rule == ip {
			return true
		}

		if m.enableCIDR && strings.Contains(rule, "/") {
			_, ipNet, err := net.ParseCIDR(rule)
			if err == nil {
				parsed := net.ParseIP(ip)
				if parsed != nil && ipNet.Contains(parsed) {
					return true
				}
			}
		}
	}

	return false
}
