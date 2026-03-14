package load_balance

import (
	"errors"
	"hash/crc32"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Hash func(data []byte) uint32

type UInt32Slice []uint32

func (s UInt32Slice) Len() int {
	return len(s)
}

func (s UInt32Slice) Less(i, j int) bool {
	return s[i] < s[j]
}

func (s UInt32Slice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type ConsistentHashBanlance struct {
	mux      sync.RWMutex
	hash     Hash
	replicas int               //复制因子
	keys     UInt32Slice       //已排序的节点hash切片
	hashMap  map[uint32]string //节点hash和key的map

	//观察主体
	conf LoadBalanceConf
}

func NewConsistentHashBanlance(replicas int, fn Hash) *ConsistentHashBanlance {
	m := &ConsistentHashBanlance{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[uint32]string),
	}
	if m.hash == nil {
		//最大2位保证是一个 2^32-1 这里
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// 验证是否为空
func (c *ConsistentHashBanlance) IsEmpty() bool {
	return len(c.keys) == 0
}

// Add 方法用来添加缓存节点，参数为节点key，比如使用IP
func (c *ConsistentHashBanlance) Add(params ...string) error {
	if len(params) == 0 {
		return errors.New("param len 1 at least")
	}
	addr := strings.TrimSpace(params[0])
	if addr == "" {
		return errors.New("param addr is empty")
	}
	c.mux.Lock()
	defer c.mux.Unlock()
	// 结合复制因子计算所有虚拟节点的hash值，并存入c.keys，同时在c.hashMap中保存hash和key映射
	for i := 0; i < c.replicas; i++ {
		hash := c.hash([]byte(strconv.Itoa(i) + addr))
		c.keys = append(c.keys, hash)
		c.hashMap[hash] = addr
	}
	// 对所有虚拟节点的hash值进行排序，方便之后进行二分查找
	sort.Sort(c.keys)
	return nil
}

// Get 方法根据给定的对象获取最靠近它的那个节点
func (c *ConsistentHashBanlance) Get(key string) (string, error) {
	c.mux.RLock()
	defer c.mux.RUnlock()
	if len(c.keys) == 0 {
		return "", ErrNoUpstream
	}
	hash := c.hash([]byte(key))

	// 通过二分查找获取最优节点
	idx := sort.Search(len(c.keys), func(i int) bool { return c.keys[i] >= hash })
	if idx == len(c.keys) {
		idx = 0
	}
	return c.hashMap[c.keys[idx]], nil
}

func (c *ConsistentHashBanlance) SetConf(conf LoadBalanceConf) {
	c.conf = conf
}

func (c *ConsistentHashBanlance) Update() {
	if conf, ok := c.conf.(*LoadBalanceCheckConf); ok {
		// fmt.Println("Update get check conf:", conf.GetConf())
		newKeys := UInt32Slice{}
		newMap := map[uint32]string{}
		for _, ip := range conf.GetConf() {
			parts := strings.Split(ip, ",")
			if len(parts) == 0 {
				continue
			}
			addr := strings.TrimSpace(parts[0])
			if addr == "" {
				continue
			}
			for i := 0; i < c.replicas; i++ {
				hash := c.hash([]byte(strconv.Itoa(i) + addr))
				newKeys = append(newKeys, hash)
				newMap[hash] = addr
			}
		}
		sort.Sort(newKeys)

		c.mux.Lock()
		c.keys = newKeys
		c.hashMap = newMap
		c.mux.Unlock()
	}
}
