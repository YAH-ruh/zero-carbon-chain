package middleware

// 全局异常恢复中间件(Recovery)
// 作用：捕获各接口未预期的 panic(如空指针、断言失败)，防止整个服务进程崩溃，
// 并统一返回 {code:500, msg, data} JSON，同时将堆栈写入分级日志便于定位。
// 变更说明(v2)：gin.Default 自带 Recovery 仅输出纯文本，替换为 JSON 化统一响应。

import (
	"runtime/debug"

	"blockchain-demo/pkg/logger"
	"blockchain-demo/pkg/response"

	"github.com/gin-gonic/gin"
)

// Recovery 全局panic恢复中间件(应放在中间件链最外层)
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("接口panic恢复 method=%s path=%s err=%v\n%s",
					c.Request.Method, c.Request.URL.Path, r, string(debug.Stack()))
				// 对外隐藏内部堆栈细节，返回统一错误码，避免服务崩溃
				response.ServerError(c, "服务器内部错误，请稍后重试")
				c.Abort()
			}
		}()
		c.Next()
	}
}

// AccessLogger 简易访问日志(记录请求方法/路径/状态码，便于答辩演示与排障)
func AccessLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if c.Writer.Status() >= 500 {
			logger.Error("请求 method=%s path=%s status=%d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
			return
		}
		if c.Writer.Status() >= 400 {
			logger.Warn("请求 method=%s path=%s status=%d", c.Request.Method, c.Request.URL.Path, c.Writer.Status())
		}
	}
}
