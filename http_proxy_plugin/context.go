package http_proxy_plugin

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
)

// ExecContext is the runtime context passed to plugins by the executor.
type ExecContext struct {
	Gin         *gin.Context
	ServiceID   int64
	ServiceName string
	PlanVersion string
	values      map[string]interface{}
}

var execContextPool = sync.Pool{
	New: func() interface{} {
		return &ExecContext{values: map[string]interface{}{}}
	},
}

func NewExecContext(c *gin.Context) *ExecContext {
	e := execContextPool.Get().(*ExecContext)
	e.reset(c)
	return e
}

func ReleaseExecContext(e *ExecContext) {
	if e == nil {
		return
	}
	e.reset(nil)
	execContextPool.Put(e)
}

func (e *ExecContext) reset(c *gin.Context) {
	e.Gin = c
	e.ServiceID = 0
	e.ServiceName = ""
	e.PlanVersion = ""

	if e.values == nil {
		e.values = map[string]interface{}{}
	} else {
		for k := range e.values {
			delete(e.values, k)
		}
	}

	if c == nil {
		return
	}
	e.ServiceName = c.GetString("service_name")
	if v, ok := c.Get("service_id"); ok {
		e.ServiceID = toInt64(v)
	}
}

func (e *ExecContext) RequestContext() context.Context {
	if e == nil || e.Gin == nil || e.Gin.Request == nil {
		return context.Background()
	}
	return e.Gin.Request.Context()
}

func (e *ExecContext) SetValue(key string, value interface{}) {
	if e == nil {
		return
	}
	if e.values == nil {
		e.values = map[string]interface{}{}
	}
	e.values[key] = value
}

func (e *ExecContext) GetValue(key string) (interface{}, bool) {
	if e == nil || e.values == nil {
		return nil, false
	}
	v, ok := e.values[key]
	return v, ok
}

func toInt64(v interface{}) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case uint64:
		return int64(n)
	case uint:
		return int64(n)
	default:
		return 0
	}
}
