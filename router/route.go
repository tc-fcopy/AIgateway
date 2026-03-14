package router

import (
	"gateway/controller"
	"gateway/middleware"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"log"
)

func InitRouter(middlewares ...gin.HandlerFunc) *gin.Engine {
	router := gin.Default()
	router.Use(middlewares...)
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
	registerSwagger(router)

	store, err := sessions.NewRedisStore(10, "tcp", "localhost:6379", "", []byte("secret"))
	if err != nil {
		log.Fatalf("sessions.NewRedisStore error: %v", err)
	}

	adminLoginRouter := router.Group("/admin_login")
	adminLoginRouter.Use(
		sessions.Sessions("admin_login", store),
		middleware.RecoveryMiddleware(),
		middleware.RequestLog(),
		middleware.TranslationMiddleware(),
	)
	{
		controller.AdminLoginRegister(adminLoginRouter)
	}

	adminRouter := router.Group("/admin")
	adminRouter.Use(
		sessions.Sessions("admin_login", store),
		middleware.RecoveryMiddleware(),
		middleware.RequestLog(),
		middleware.SessionAuthMiddleware(),
		middleware.TranslationMiddleware(),
	)
	{
		controller.AdminRegister(adminRouter)

		aiRouter := adminRouter.Group("/ai")
		controller.AIConsumerRegister(aiRouter)
		controller.AIQuotaRegister(aiRouter)
		controller.AIServiceConfigRegister(aiRouter)

		pipelineRouter := adminRouter.Group("/pipeline")
		controller.PipelineRegister(pipelineRouter)
	}

	serviceRouter := router.Group("/service")
	serviceRouter.Use(
		sessions.Sessions("admin_login", store),
		middleware.RecoveryMiddleware(),
		middleware.RequestLog(),
		middleware.SessionAuthMiddleware(),
		middleware.TranslationMiddleware(),
	)
	{
		controller.ServiceRegister(serviceRouter)
	}

	return router
}
