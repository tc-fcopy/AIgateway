package load_balance

import (
	"errors"
	"math/rand"
	"strings"
	"sync"
)

type RandomBalance struct {
	mu       sync.Mutex
	curIndex int
	rss      []string
	//观察主体
	conf LoadBalanceConf
}

func (r *RandomBalance) Add(params ...string) error {
	if len(params) == 0 {
		return errors.New("param len 1 at least")
	}
	addr := strings.TrimSpace(params[0])
	if addr == "" {
		return errors.New("param addr is empty")
	}
	return r.addLocked(addr)
}

func (r *RandomBalance) addLocked(addr string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rss = append(r.rss, addr)
	return nil
}

func (r *RandomBalance) Next() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.rss) == 0 {
		return ""
	}
	r.curIndex = rand.Intn(len(r.rss))
	return r.rss[r.curIndex]
}

func (r *RandomBalance) Get(key string) (string, error) {
	addr := r.Next()
	if addr == "" {
		return "", ErrNoUpstream
	}
	return addr, nil
}

func (r *RandomBalance) SetConf(conf LoadBalanceConf) {
	r.conf = conf
}

func (r *RandomBalance) Update() {
	if conf, ok := r.conf.(*LoadBalanceCheckConf); ok {
		// fmt.Println("Update get check conf:", conf.GetConf())
		var nodes []string
		for _, ip := range conf.GetConf() {
			parts := strings.Split(ip, ",")
			if len(parts) == 0 {
				continue
			}
			addr := strings.TrimSpace(parts[0])
			if addr != "" {
				nodes = append(nodes, addr)
			}
		}
		r.mu.Lock()
		r.rss = nodes
		r.curIndex = 0
		r.mu.Unlock()
	}
}
