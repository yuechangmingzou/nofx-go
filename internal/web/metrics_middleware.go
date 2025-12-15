package web

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/nofx-go/internal/metrics"
)

// metricsMiddleware 指标收集中间件
func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		// 记录HTTP请求指标
		metrics.RecordHTTPRequest(c.Request.URL.Path, status, latency)
	}
}

