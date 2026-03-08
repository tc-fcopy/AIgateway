package loadbalancer

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"gateway/ai_gateway/common"
)

var (
	// LoadBalancer 全局负载均衡器
	LoadBalancer = NewGlobalLeastRequest()
)

// Backend 后端节点
type Backend struct {
	ID       string        `json:"id"`         // 节点ID
	Address  string        `json:"address"`    // 节点地址
	Weight   int           `json:"weight"`     // 权重
	Requests int64         `json:"requests"`  // 当前请求数（原子操作）
	Enabled  bool          `json:"enabled"`   // 是否启用
	LastUsed time.Time     `json:"last_used"` // 最后使用时间
}

// GlobalLeastRequest 全局最小请求数负载均衡器
type GlobalLeastRequest struct {
	backends           map[string]*Backend
	lock               sync.RWMutex
	enable             bool
	healthCheckInterval time.Duration
	healthCheckTicker  *time.Ticker
	stopChan           chan struct{}
}

// NewGlobalLeastRequest 创建负载均衡器
func NewGlobalLeastRequest() *GlobalLeastRequest {
	return &GlobalLeastRequest{
		backends: make(map[string]*Backend),
		enable:   true,
	}
}

// AddBackend 添加后端节点
func (g *GlobalLeastRequest) AddBackend(backend *Backend) error {
	if backend == nil {
		return fmt.Errorf("backend is ail")
	}

	if backend.ID == "" {
		return fmt.Errorf("backend id is empty")
	}

	if backend.Address == "" {
		return fmt.Errorf("backend address is empty")
	}

	g.lock.Lock()
	defer g.lock.Unlock()

	// 检查是否已存在
	if _, exists := g.backends[backend.ID]; exists {
		return fmt.Errorf("backend already exists: %s", backend.ID)
	}

	// 初始化原子计数
	backend.Requests = 0
	backend.Enabled = true
	backend.LastUsed = time.Time{}

	g.backends[backend.ID] = backend
	return nil
}

// RemoveBackend 移除后端节点
func (g *GlobalLeastRequest) RemoveBackend(backendID string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	delete(g.backends, backendID)
}

// UpdateBackend 更新后端节点
func (g *GlobalLeastRequest) UpdateBackend(backend *Backend) error {
	if backend == nil {
		return fmt.Errorf("backend is nil")
	}

	g.lock.Lock()
	defer g.lock.Unlock()

	existing, exists := g.backends[backend.ID]
	if !exists {
		return fmt.Errorf("backend not found: %s", backend.ID)
	}

	// 只更新可修改的字段
	if backend.Address != "" {
		existing.Address = backend.Address
	}
	if backend.Weight > 0 {
		existing.Weight = backend.Weight
	}
	existing.Enabled = backend.Enabled

	return nil
}

// SelectBackend 选择后端节点（最小请求数算法）
func (g *GlobalLeastRequest) SelectBackend() (*Backend, error) {
	if !g.enable {
		return nil, fmt.Errorf("load balancer is disabled")
	}

	g.lock.RLock()
	defer g.lock.RUnlock()

	if len(g.backends) == 0 {
		return nil, fmt.Errorf("no available backends")
	}

	// 查找请求数最少且启用的后端
	var selectedBackend *Backend
	minRequests := int64(-1)

	for _, backend := range g.backends {
		if !backend.Enabled {
			continue
		}

		currentRequests := atomic.LoadInt64(&backend.Requests)

		// 优先选择请求数最少的
		if minRequests < 0 || currentRequests < minRequests {
			minRequests = currentRequests
			selectedBackend = backend
		}
	}

	if selectedBackend == nil {
		return nil, fmt.Errorf("no enabled backend available")
	}

	// 原子操作增加请求数
	atomic.AddInt64(&selectedBackend.Requests, 1)

	// 记录最后使用时间
	selectedBackend.LastUsed = time.Now()

	// 使用Redis记录负载均衡信息（可选）
	g.recordBackendUsage(selectedBackend.ID)

	return selectedBackend, nil
}

// GetBackend 获取指定后端节点
func (g *GlobalLeastRequest) GetBackend(backendID string) (*Backend, error) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	backend, exists := g.backends[backendID]
	if !exists {
		return nil, fmt.Errorf("backend not found: %s", backendID)
	}

	return backend, nil
}

// GetAllBackends 获取所有后端节点
func (g *GlobalLeastRequest) GetAllBackends() []*Backend {
	g.lock.RLock()
	defer g.lock.RUnlock()

	backends := make([]*Backend, 0, len(g.backends))
	for _, backend := range g.backends {
		backends = append(backends, backend)
	}
	return backends
}

// ReleaseBackend 释放后端节点（请求完成后调用）
func (g *GlobalLeastRequest) ReleaseBackend(backendID string) {
	g.lock.RLock()
	defer g.lock.RUnlock()

	backend, exists := g.backends[backendID]
	if !exists {
		return
	}

	// 原子操作减少请求数
	atomic.AddInt64(&backend.Requests, -1)
}

// recordBackendUsage 记录后端使用到Redis
func (g *GlobalLeastRequest) recordBackendUsage(backendID string) {
	// 使用Redis记录使用统计信息
	redisKey := common.BuildLoadBalancerKey(backendID)

	// 简单记录：记录时间戳
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano())

	// Redis原子记录：HINCRBY或类似
	// 这里简化处理，实际应该使用更复杂的统计逻辑
	// TODO: 实现基于Redis的使用统计
	_ = redisKey
	_ = timestamp
}

// GetBackendStats 获取后端统计信息
func (g *GlobalLeastRequest) GetBackendStats() map[string]interface{} {
	g.lock.RLock()
	defer g.lock.RUnlock()

	stats := make(map[string]interface{})
	for id, backend := range g.backends {
		stats[id] = map[string]interface{}{
			"address":      backend.Address,
			"weight":       backend.Weight,
			"requests":     atomic.LoadInt64(&backend.Requests),
			"enabled":      backend.Enabled,
			"last_used":    backend.LastUsed.Format(time.RFC3339),
		}
	}
	return stats
}

// Enable 启用负载均衡
func (g *GlobalLeastRequest) Enable() {
	g.lock.Lock()
	defer g.lock.Unlock()
	g.enable = true
}

// Disable 禁用负载均衡
func (g *GlobalLeastRequest) Disable() {
	g.lock.Lock()
	defer g.lock.Unlock()
	g.enable = false
}

// IsEnabled 检查是否启用
func (g *GlobalLeastRequest) IsEnabled() bool {
	g.lock.RLock()
	defer g.lock.RUnlock()
	return g.enable
}

// StartHealthCheck 启动健康检查
func (g *GlobalLeastRequest) StartHealthCheck(interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second // 默认30秒
	}

	g.lock.Lock()
	defer g.lock.Unlock()

	// 停止已有的检查
	if g.healthCheckTicker != nil {
		g.healthCheckTicker.Stop()
	}

	g.healthCheckInterval = interval
	g.healthCheckTicker = time.NewTicker(interval)
	g.stopChan = make(chan struct{})

	// 启动检查协程
	go g.healthCheckLoop()
}

// StopHealthCheck 停止健康检查
func (g *GlobalLeastRequest) StopHealthCheck() {
	g.lock.Lock()
	defer g.lock.Unlock()

	if g.healthCheckTicker != nil {
		g.healthCheckTicker.Stop()
		close(g.stopChan)
		g.healthCheckTicker = nil
	}
}

// healthCheckLoop 健康检查循环
func (g *GlobalLeastRequest) healthCheckLoop() {
	for {
		select {
		case <-g.healthCheckTicker.C:
			g.checkBackendHealth()
		case <-g.stopChan:
			return
		}
	}
}

// checkBackendHealth 检查后端健康状态
func (g *GlobalLeastRequest) checkBackendHealth() {
	// TODO: 实现实际的健康检查逻辑
	// 这里只是示例，实际应该：
	// 1. 对每个后端发送健康检查请求
	// 2. 根据响应判断后端是否健康
	// 3. 自动标记不健康的后端为禁用状态

	// 示例：检查所有后端
	g.lock.RLock()
	backends := make([]*Backend, 0, len(g.backends))
	for _, backend := range g.backends {
		backends = append(backends, backend)
	}
	g.lock.RUnlock()

	for _, backend := range backends {
		if backend.Enabled {
			// 模拟健康检查（实际应该发送HTTP请求）
			healthy := g.isBackendHealthy(backend.Address)

			if !healthy {
				// 标记为禁用
				backend.Enabled = false
				fmt.Printf("[Load Balancer] Backend %s marked as unhealthy\n", backend.ID)
			}
		}
	}
}

// isBackendHealthy 检查后端是否健康
func (g *GlobalLeastRequest) isBackendHealthy(address string) bool {
	// TODO: 实现实际的健康检查
	// 这里返回true作为默认值
	return true
}

// Reset 重置负载均衡器
func (g *GlobalLeastRequest) Reset() {
	g.lock.Lock()
	defer g.lock.Unlock()

	// 停止健康检查
	if g.healthCheckTicker != nil {
		g.healthCheckTicker.Stop()
		close(g.stopChan)
		g.healthCheckTicker = nil
	}

	// 清空后端节点
	g.backends = make(map[string]*Backend)
	g.enable = true
}
