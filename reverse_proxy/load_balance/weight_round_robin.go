package load_balance

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type WeightRoundRobinBalance struct {
	mu       sync.Mutex
	curIndex int
	rss      []*WeightNode
	rsw      []int
	//观察主体
	conf LoadBalanceConf
}

type WeightNode struct {
	addr            string
	weight          int //权重值
	currentWeight   int //节点当前权重
	effectiveWeight int //有效权重
}

func (r *WeightRoundRobinBalance) Add(params ...string) error {
	if len(params) != 2 {
		return errors.New("param len need 2")
	}
	addr := strings.TrimSpace(params[0])
	if addr == "" {
		return errors.New("param addr is empty")
	}
	parInt, err := strconv.ParseInt(strings.TrimSpace(params[1]), 10, 64)
	if err != nil {
		return err
	}
	if parInt <= 0 {
		return errors.New("param weight must be positive")
	}
	node := &WeightNode{addr: addr, weight: int(parInt)}
	node.effectiveWeight = node.weight

	r.mu.Lock()
	defer r.mu.Unlock()
	r.rss = append(r.rss, node)
	return nil
}

func (r *WeightRoundRobinBalance) Next() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.rss) == 0 {
		return ""
	}
	total := 0
	var best *WeightNode
	for i := 0; i < len(r.rss); i++ {
		w := r.rss[i]
		//step 1 统计所有有效权重之和
		total += w.effectiveWeight

		//step 2 变更节点临时权重=节点临时权重+节点有效权重
		w.currentWeight += w.effectiveWeight

		//step 3 有效权重默认与权重相同，通信异常时-1，通信成功+1，直到恢复到weight大小
		if w.effectiveWeight < w.weight {
			w.effectiveWeight++
		}
		//step 4 选择最大临时权重节点
		if best == nil || w.currentWeight > best.currentWeight {
			best = w
		}
	}
	if best == nil {
		return ""
	}
	//step 5 变更临时权重=临时权重-有效权重之和
	best.currentWeight -= total
	return best.addr
}

func (r *WeightRoundRobinBalance) Get(key string) (string, error) {
	addr := r.Next()
	if addr == "" {
		return "", ErrNoUpstream
	}
	return addr, nil
}

func (r *WeightRoundRobinBalance) SetConf(conf LoadBalanceConf) {
	r.conf = conf
}

func (r *WeightRoundRobinBalance) Update() {
	if conf, ok := r.conf.(*LoadBalanceCheckConf); ok {
		fmt.Println("WeightRoundRobinBalance get check conf:", conf.GetConf())
		var nodes []*WeightNode
		for _, ip := range conf.GetConf() {
			parts := strings.Split(ip, ",")
			if len(parts) == 0 {
				continue
			}
			addr := strings.TrimSpace(parts[0])
			if addr == "" {
				continue
			}
			weight := 1
			if len(parts) > 1 {
				if w, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && w > 0 {
					weight = w
				}
			}
			node := &WeightNode{addr: addr, weight: weight, effectiveWeight: weight}
			nodes = append(nodes, node)
		}
		r.mu.Lock()
		r.rss = nodes
		r.curIndex = 0
		r.mu.Unlock()
	}
}
