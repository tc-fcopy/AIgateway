package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	aiconfig "gateway/ai_gateway/config"
	"gateway/dao"
	"gateway/http_proxy_pipeline"
	"gateway/http_proxy_plugin"
	"gateway/http_proxy_router"
	"github.com/gin-gonic/gin"
)

func main() {
	var port int
	var duration int

	flag.IntVar(&port, "p", 8080, "Port to listen on")
	flag.IntVar(&duration, "d", 30, "Duration of test in seconds")
	flag.Parse()

	// 初始化
	gin.SetMode(gin.ReleaseMode)
	http_proxy_plugin.RegisterBuiltinPluginsTo(http_proxy_plugin.GlobalRegistry)

	// 配置
	initAIConfig()

	// 创建测试服务器
	router := gin.New()
	router.Use(gin.Recovery())

	// 简单的测试路由 - 包含我们的中间件
	router.GET("/test",
		// 设置模拟的service（实际网关通过路由参数获取）
		func(c *gin.Context) {
			testService := &dao.ServiceDetail{
				Info: &dao.ServiceInfo{
					ID:          1001,
					ServiceName: "test-service",
					LoadType:    1,
				},
			}
			c.Set("service", testService)
		},
		http_proxy_pipeline.PipelinePlanMiddleware(),
		http_proxy_pipeline.PipelineExecutorMiddleware(),
		// 最终响应
		func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"message": "Test response",
			})
		},
	)

	// 启动服务器
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	fmt.Printf("🚀 Test server is running on http://localhost:%d\n", port)
	fmt.Printf("📊 Test route: /test (GET)\n")
	fmt.Printf("⏱️  You have %d seconds to run your load test...\n", duration)
	fmt.Printf("💡 Suggestions:\n")
	fmt.Printf("  1. Use wrk: wrk -t8 -c100 -d%d http://localhost:%d/test\n", duration, port)
	fmt.Printf("  2. Use hey: hey -n 100000 -c 100 -z %ds http://localhost:%d/test\n", duration, port)
	fmt.Printf("  3. Use ab: ab -n 10000 -c 100 http://localhost:%d/test\n", port)

	// 等待测试时间
	<-time.After(time.Duration(duration) * time.Second)

	// 优雅关闭
	fmt.Println("\n🔴 Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	fmt.Println("✅ Server exited normally")
}

func initAIConfig() {
	aiconfig.AIConfManager.SetConfig(&aiconfig.AIConfig{
		Enable:             true,
		ApplyToAllServices: boolPtr(true),
		DefaultService: aiconfig.AIServiceConfig{
			EnableKeyAuth:         true,
			EnableIPRestriction:   true,
			EnableTokenRateLimit:  true,
			EnableQuota:           true,
			EnableCORS:            true,
			EnableModelRouter:     true,
			EnablePromptDecorator: true,
			EnableCache:           true,
			EnableLoadBalancer:    true,
			EnableObservability:   true,
		},
		Pipeline: aiconfig.PipelineConfig{
			StrictDependency: true,
		},
	})
}

func boolPtr(v bool) *bool {
	p := v
	return &p
}
