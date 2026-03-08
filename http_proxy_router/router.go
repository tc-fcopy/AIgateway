package http_proxy_router

import (
	"gateway/http_proxy_middleware"
	"gateway/http_proxy_pipeline"
	"github.com/gin-gonic/gin"
)

func InitRouter(middlewares ...gin.HandlerFunc) *gin.Engine {
	router := gin.New()
	router.Use(middlewares...)

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	router.Use(
		// Step 1: service resolving + dynamic plan building.
		http_proxy_middleware.HTTPAccessModeMiddleware(),
		http_proxy_pipeline.PipelinePlanMiddleware(),
		http_proxy_pipeline.PipelineExecutorMiddleware(),
	)

	return router
}
