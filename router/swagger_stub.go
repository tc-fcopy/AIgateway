package router

import "github.com/gin-gonic/gin"

// registerSwagger is a no-op in the default build to keep swagger out of proxy process.
func registerSwagger(_ *gin.Engine) {}
