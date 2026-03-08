package http_proxy_plugin

import (
	"errors"

	"github.com/gin-gonic/gin"
)

// AdapterSpec defines metadata for wrapping an existing middleware as a plugin.
type AdapterSpec struct {
	Name     string
	Phase    Phase
	Priority int
	Requires []string
	Enabled  func(*ExecContext) bool
}

// MiddlewareAdapter adapts gin middleware into Plugin contract.
//
// Note: this is a migration helper; legacy middlewares that rely on deep c.Next()
// chaining semantics should be migrated to native plugins before executor cut-over.
type MiddlewareAdapter struct {
	spec    AdapterSpec
	handler gin.HandlerFunc
}

func NewMiddlewareAdapter(spec AdapterSpec, handler gin.HandlerFunc) (*MiddlewareAdapter, error) {
	if spec.Name == "" {
		return nil, errors.New("adapter plugin name is required")
	}
	if handler == nil {
		return nil, errors.New("adapter handler is nil")
	}
	copied := make([]string, len(spec.Requires))
	copy(copied, spec.Requires)
	spec.Requires = copied
	return &MiddlewareAdapter{spec: spec, handler: handler}, nil
}

func (a *MiddlewareAdapter) Name() string { return a.spec.Name }

func (a *MiddlewareAdapter) Phase() Phase { return a.spec.Phase }

func (a *MiddlewareAdapter) Priority() int { return a.spec.Priority }

func (a *MiddlewareAdapter) Requires() []string {
	copied := make([]string, len(a.spec.Requires))
	copy(copied, a.spec.Requires)
	return copied
}

func (a *MiddlewareAdapter) Enabled(ctx *ExecContext) bool {
	if a.spec.Enabled == nil {
		return true
	}
	return a.spec.Enabled(ctx)
}

func (a *MiddlewareAdapter) Handler() gin.HandlerFunc {
	return a.handler
}

func (a *MiddlewareAdapter) Execute(ctx *ExecContext) Result {
	if ctx == nil || ctx.Gin == nil {
		return Abort(errors.New("execution context is nil"))
	}

	beforeStatus := ctx.Gin.Writer.Status()
	a.handler(ctx.Gin)
	if ctx.Gin.IsAborted() {
		status := ctx.Gin.Writer.Status()
		if status == 0 {
			status = beforeStatus
		}
		return AbortWithStatus(status, nil)
	}
	return Continue()
}
