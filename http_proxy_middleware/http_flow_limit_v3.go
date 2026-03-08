package http_proxy_middleware

import (
	"errors"
	"fmt"
	"gateway/dao"
	"gateway/middleware"
	"gateway/public"
	"github.com/gin-gonic/gin"
)

func HTTPFlowLimitMiddlewareV3() gin.HandlerFunc {
	return func(c *gin.Context) {
		serverInterface, ok := c.Get("service")
		if !ok {
			middleware.ResponseError(c, 2001, errors.New("service not found"))
			c.Abort()
			return
		}
		serviceDetail := serverInterface.(*dao.ServiceDetail)

		fmt.Printf("服务限流配置：%d，客户端IP限流配置：%d\n",
			serviceDetail.AccessControl.ServiceFlowLimit,
			serviceDetail.AccessControl.ClientIPFlowLimit)
		if serviceDetail.AccessControl.ServiceFlowLimit != 0 {
			fmt.Println("获取限流器：", float64(serviceDetail.AccessControl.ServiceFlowLimit))
			serviceLimiter, err := public.FlowLimiterHandler.GetLimiter(
				public.FlowServicePrefix+serviceDetail.Info.ServiceName,
				float64(serviceDetail.AccessControl.ServiceFlowLimit))
			if err != nil {
				middleware.ResponseError(c, 5001, err)
				c.Abort()
				return
			}
			if !serviceLimiter.Allow() {
				fmt.Println("请求被限流！")
				middleware.ResponseError(c, 5002, errors.New(fmt.Sprintf("service flow limit %v", serviceDetail.AccessControl.ServiceFlowLimit)))
				c.Abort()
				return
			}
			fmt.Println("请求通过")
		}

		if serviceDetail.AccessControl.ClientIPFlowLimit > 0 {
			clientLimiter, err := public.FlowLimiterHandler.GetLimiter(
				public.FlowServicePrefix+serviceDetail.Info.ServiceName+"_"+c.ClientIP(),
				float64(serviceDetail.AccessControl.ClientIPFlowLimit))
			if err != nil {
				middleware.ResponseError(c, 5003, err)
				c.Abort()
				return
			}
			if !clientLimiter.Allow() {
				fmt.Println("请求被限流！")
				middleware.ResponseError(c, 5002, errors.New(fmt.Sprintf("%v flow limit %v", c.ClientIP(), serviceDetail.AccessControl.ClientIPFlowLimit)))
				c.Abort()
				return
			}
			fmt.Println("请求通过限流")
		}
		c.Next()
	}
}
