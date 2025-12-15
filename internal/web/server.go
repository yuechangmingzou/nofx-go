package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/nofx-go/internal/config"
	"github.com/yourusername/nofx-go/internal/exchange"
	"github.com/yourusername/nofx-go/internal/utils"
	"go.uber.org/zap"
)

// Server Web服务器
type Server struct {
	engine   *gin.Engine
	config   *config.Config
	logger   *zap.SugaredLogger
	exchange *exchange.BinanceExchange
	redis    utils.RedisClient
}

var globalServer *Server

// GetServer 获取Web服务器实例（单例）
func GetServer() *Server {
	if globalServer == nil {
		globalServer = &Server{
			engine:   gin.Default(),
			config:   config.Get(),
			logger:   utils.GetLogger("web"),
			exchange: exchange.GetBinanceExchange(),
			redis:    utils.GetRedisClient(),
		}
		globalServer.setupRoutes()
	}
	return globalServer
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 添加中间件
	s.engine.Use(s.recoveryMiddleware())
	s.engine.Use(s.loggerMiddleware())
	s.engine.Use(s.metricsMiddleware())

	// 静态文件
	staticDir := s.getStaticDir()
	if staticDir != "" {
		s.engine.Static("/static", staticDir)
	}

	// 健康检查（无需认证）
	s.engine.GET("/healthz", s.handleHealthz)
	s.engine.GET("/readyz", s.handleReadyz)

	// API路由组（需要认证）
	api := s.engine.Group("/api")
	api.Use(s.basicAuthMiddleware())
	{
		// 状态
		api.GET("/status", s.handleStatus)
		api.GET("/market-data", s.handleMarketData)

		// AI模式
		api.GET("/ai-mode", s.handleGetAIMode)
		api.POST("/ai-mode", s.handleSetAIMode)

		// AI提示词
		api.GET("/ai-prompt", s.handleGetAIPrompt)
		api.POST("/ai-prompt", s.handleSetAIPrompt)
		api.DELETE("/ai-prompt", s.handleDeleteAIPrompt)

		// 运行时配置
		api.GET("/runtime-config", s.handleGetRuntimeConfig)
		api.POST("/runtime-config", s.handleSetRuntimeConfig)
		api.DELETE("/runtime-config", s.handleDeleteRuntimeConfig)
		api.GET("/runtime-config/audit", s.handleRuntimeConfigAudit)

		// WebSocket token
		api.GET("/ws-token", s.handleWSToken)

		// 余额和持仓
		api.GET("/balance", s.handleBalance)
		api.GET("/positions", s.handlePositions)
		api.GET("/equity", s.handleEquity)

		// 历史
		api.GET("/history", s.handleHistory)
		api.GET("/latest-ai-decision", s.handleLatestAIDecision)

		// 扫描的币种
		api.GET("/scanned-symbols", s.handleScannedSymbols)
	}

	// WebSocket
	s.engine.GET("/ws", s.handleWebSocket)

	// 首页
	s.engine.GET("/", s.handleIndex)
}

// Run 启动服务器
func (s *Server) Run(ctx context.Context) error {
	port := s.config.WebPort
	if port <= 0 {
		port = 8000
	}

	addr := fmt.Sprintf(":%d", port)
	s.logger.Infow("Web服务器启动",
		"addr", addr,
		"static_dir", s.getStaticDir(),
	)

	server := &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 优雅关闭
	go func() {
		<-ctx.Done()
		s.logger.Info("Web服务器正在关闭...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// getStaticDir 获取静态文件目录
func (s *Server) getStaticDir() string {
	staticDir := s.config.WebStaticDir
	if staticDir == "" {
		// 默认路径
		projectRoot := os.Getenv("PROJECT_ROOT")
		if projectRoot == "" {
			projectRoot = "."
		}
		staticDir = filepath.Join(projectRoot, "web", "static")
	}

	// 检查目录是否存在
	if info, err := os.Stat(staticDir); err == nil && info.IsDir() {
		return staticDir
	}

	return ""
}

// basicAuthMiddleware BasicAuth中间件
func (s *Server) basicAuthMiddleware() gin.HandlerFunc {
	return gin.BasicAuth(gin.Accounts{
		s.config.WebBasicAuthUser: s.config.WebBasicAuthPass,
	})
}

// recoveryMiddleware 恢复中间件
func (s *Server) recoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				s.logger.Errorw("请求处理panic",
					"error", err,
					"path", c.Request.URL.Path,
				)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal_server_error",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// loggerMiddleware 日志中间件
func (s *Server) loggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if status >= 500 {
			s.logger.Warnw("HTTP请求",
				"status", status,
				"method", c.Request.Method,
				"path", path,
				"latency", latency,
				"ip", c.ClientIP(),
			)
		}
	}
}

// handleHealthz 健康检查
func (s *Server) handleHealthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleReadyz 就绪检查
func (s *Server) handleReadyz(c *gin.Context) {
	// 检查Redis连接
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.redis.Ping(ctx).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready", "error": "redis_unavailable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

// handleIndex 首页
func (s *Server) handleIndex(c *gin.Context) {
	templatePath := s.config.WebDashboardTemplate
	if templatePath == "" {
		projectRoot := os.Getenv("PROJECT_ROOT")
		if projectRoot == "" {
			projectRoot = "."
		}
		templatePath = filepath.Join(projectRoot, "web", "templates", "dashboard.html")
	}

	// 读取模板文件
	data, err := os.ReadFile(templatePath)
	if err != nil {
		c.String(http.StatusNotFound, "Dashboard template not found")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}
