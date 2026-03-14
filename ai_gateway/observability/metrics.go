package observability

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

// Metrics stores minimal in-memory counters and can expose Prometheus text format.
type Metrics struct {
	totalRequests uint64 // 总请求数
	totalErrors   uint64 // 总错误数

	mu          sync.RWMutex      // 读写锁，保证并发安全
	byService   map[string]uint64 // 按服务统计请求数
	byModel     map[string]uint64 // 按模型统计请求数
	totalTokens uint64            // 总消耗 Token 数
}

var (
	GlobalMetrics = NewMetrics()
	metricsOnce   sync.Once
)

func NewMetrics() *Metrics {
	return &Metrics{
		byService: map[string]uint64{},
		byModel:   map[string]uint64{},
	}
}

func (m *Metrics) RecordRequest(service, model string, statusCode int, totalTokens int64) {
	//总请求数 +1（原子操作，高并发安全）
	//状态码 ≥400 算错误，错误数 +1
	//累计消耗的总 Token 数
	//按 service、model 维度分别统计请求次数
	atomic.AddUint64(&m.totalRequests, 1)
	if statusCode >= 400 {
		atomic.AddUint64(&m.totalErrors, 1)
	}
	if totalTokens > 0 {
		atomic.AddUint64(&m.totalTokens, uint64(totalTokens))
	}

	m.mu.Lock()
	if service != "" {
		m.byService[service]++
	}
	if model != "" {
		m.byModel[model]++
	}
	m.mu.Unlock()
}

func (m *Metrics) RenderPrometheus() string {
	var b strings.Builder
	b.WriteString("# TYPE ai_gateway_requests_total counter\n")
	b.WriteString(fmt.Sprintf("ai_gateway_requests_total %d\n", atomic.LoadUint64(&m.totalRequests)))
	b.WriteString("# TYPE ai_gateway_errors_total counter\n")
	b.WriteString(fmt.Sprintf("ai_gateway_errors_total %d\n", atomic.LoadUint64(&m.totalErrors)))
	b.WriteString("# TYPE ai_gateway_tokens_total counter\n")
	b.WriteString(fmt.Sprintf("ai_gateway_tokens_total %d\n", atomic.LoadUint64(&m.totalTokens)))

	m.mu.RLock()
	services := make([]string, 0, len(m.byService))
	for svc := range m.byService {
		services = append(services, svc)
	}
	sort.Strings(services)
	for _, svc := range services {
		b.WriteString(fmt.Sprintf("ai_gateway_requests_by_service_total{service=\"%s\"} %d\n", svc, m.byService[svc]))
	}

	models := make([]string, 0, len(m.byModel))
	for model := range m.byModel {
		models = append(models, model)
	}
	sort.Strings(models)
	for _, model := range models {
		b.WriteString(fmt.Sprintf("ai_gateway_requests_by_model_total{model=\"%s\"} %d\n", model, m.byModel[model]))
	}
	m.mu.RUnlock()

	return b.String()
}

// StartMetricsEndpoint exposes /metrics once.
func StartMetricsEndpoint(port int) {
	if port <= 0 {
		port = 9090
	}

	metricsOnce.Do(func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain; version=0.0.4")
			_, _ = w.Write([]byte(GlobalMetrics.RenderPrometheus()))
		})

		addr := fmt.Sprintf(":%d", port)
		go func() {
			if err := http.ListenAndServe(addr, mux); err != nil {
				log.Printf("[ai-metrics] listen error: %v", err)
			}
		}()
	})
}
