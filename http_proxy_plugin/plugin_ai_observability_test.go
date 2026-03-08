package http_proxy_plugin

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	aiconfig "gateway/ai_gateway/config"
	aigwctx "gateway/ai_gateway/context"
	"gateway/ai_gateway/token"
	"github.com/gin-gonic/gin"
)

type mockObservabilityMetrics struct {
	records []struct {
		service string
		model   string
		status  int
		tokens  int64
	}
}

func (m *mockObservabilityMetrics) RecordRequest(service, model string, statusCode int, totalTokens int64) {
	m.records = append(m.records, struct {
		service string
		model   string
		status  int
		tokens  int64
	}{
		service: service,
		model:   model,
		status:  statusCode,
		tokens:  totalTokens,
	})
}

type mockObservabilityLogger struct {
	requestIDSet     string
	requestIDCleared bool
	infoMessages     []string
	warnMessages     []string
	fieldsSeen       []map[string]interface{}
}

func (l *mockObservabilityLogger) SetRequestID(requestID string) {
	l.requestIDSet = requestID
}

func (l *mockObservabilityLogger) WithFields(fields map[string]interface{}) observabilityEntryLike {
	copied := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		copied[k] = v
	}
	l.fieldsSeen = append(l.fieldsSeen, copied)
	return &mockObservabilityEntry{logger: l}
}

func (l *mockObservabilityLogger) ClearRequestID() {
	l.requestIDCleared = true
}

type mockObservabilityEntry struct {
	logger *mockObservabilityLogger
}

func (e *mockObservabilityEntry) Info(format string, args ...interface{}) {
	e.logger.infoMessages = append(e.logger.infoMessages, fmt.Sprintf(format, args...))
}

func (e *mockObservabilityEntry) Warn(format string, args ...interface{}) {
	e.logger.warnMessages = append(e.logger.warnMessages, fmt.Sprintf(format, args...))
}

func TestAIObservabilityPluginSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIObservabilityPluginTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableObservability: true,
		},
	})

	mockLogger := &mockObservabilityLogger{}
	mockMetrics := &mockObservabilityMetrics{}
	aiObservabilityLoggerGetter = func() observabilityLoggerLike { return mockLogger }
	aiObservabilityMetricsGetter = func() observabilityMetricsLike { return mockMetrics }

	var requestIDInHandler string
	var startTimeInHandler int64

	r := gin.New()
	r.Use(NewAIObservabilityPlugin().Handler())
	r.GET("/", func(c *gin.Context) {
		c.Set("service_name", "svc-obs")
		c.Set(aigwctx.ModelKey, "gpt-4o")
		c.Set(aigwctx.ConsumerNameKey, "consumer-obs")
		c.Set(aigwctx.TokenUsageKey, &token.TokenUsage{TotalTokens: 42})
		requestIDInHandler = c.GetString("request_id")
		startTimeInHandler = c.GetInt64(aigwctx.StartTimeKey)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if requestIDInHandler == "" {
		t.Fatalf("expected request_id in context")
	}
	if startTimeInHandler <= 0 {
		t.Fatalf("expected start time in context")
	}
	if mockLogger.requestIDSet == "" || !mockLogger.requestIDCleared {
		t.Fatalf("expected request id set and cleared, got set=%q cleared=%v", mockLogger.requestIDSet, mockLogger.requestIDCleared)
	}
	if len(mockLogger.infoMessages) < 2 {
		t.Fatalf("expected at least 2 info logs, got %d", len(mockLogger.infoMessages))
	}
	if len(mockLogger.warnMessages) != 0 {
		t.Fatalf("expected no warn logs for success, got %d", len(mockLogger.warnMessages))
	}
	if len(mockMetrics.records) != 1 {
		t.Fatalf("expected 1 metric record, got %d", len(mockMetrics.records))
	}
	rec := mockMetrics.records[0]
	if rec.service != "svc-obs" || rec.model != "gpt-4o" || rec.status != http.StatusOK || rec.tokens != 42 {
		t.Fatalf("unexpected metric record: %+v", rec)
	}
}

func TestAIObservabilityPluginErrorStatusWarn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	restore := prepareAIObservabilityPluginTestState()
	defer restore()

	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable: true,
		DefaultService: aiconfig.AIServiceConfig{
			EnableObservability: true,
		},
	})

	mockLogger := &mockObservabilityLogger{}
	mockMetrics := &mockObservabilityMetrics{}
	aiObservabilityLoggerGetter = func() observabilityLoggerLike { return mockLogger }
	aiObservabilityMetricsGetter = func() observabilityMetricsLike { return mockMetrics }

	r := gin.New()
	r.Use(NewAIObservabilityPlugin().Handler())
	r.GET("/", func(c *gin.Context) {
		c.Set("service_name", "svc-err")
		c.Set(aigwctx.ModelKey, "gpt-4o-mini")
		c.Status(http.StatusInternalServerError)
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	if len(mockLogger.warnMessages) != 1 {
		t.Fatalf("expected 1 warn log for error status, got %d", len(mockLogger.warnMessages))
	}
	if len(mockMetrics.records) != 1 {
		t.Fatalf("expected 1 metric record, got %d", len(mockMetrics.records))
	}
	if mockMetrics.records[0].status != http.StatusInternalServerError {
		t.Fatalf("expected metric status 500, got %d", mockMetrics.records[0].status)
	}
}

func prepareAIObservabilityPluginTestState() func() {
	prevConf := aiconfig.AIConfManager.GetConfig()
	prevLoggerGetter := aiObservabilityLoggerGetter
	prevMetricsGetter := aiObservabilityMetricsGetter
	return func() {
		aiconfig.AIConfManager.SetConfig(prevConf)
		aiObservabilityLoggerGetter = prevLoggerGetter
		aiObservabilityMetricsGetter = prevMetricsGetter
	}
}
