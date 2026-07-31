package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"

	"github.com/example/gin-high-performance/internal/config"
	"github.com/example/gin-high-performance/internal/logger"
	"github.com/example/gin-high-performance/pkg/jwt"
	"github.com/example/gin-high-performance/pkg/ws"
)

func main() {
	log.Println("Starting WebSocket server...")

	// 1. 初始化配置与日志
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	logger.Init(cfg.LogLevel)

	// 2. 现代化的信号上下文监听
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 3. 初始化 Hub
	hub := ws.NewHub()

	// 4. 注册路由
	router := setupWSRouter(cfg, hub)

	srv := &http.Server{
		Addr:    ":" + cfg.WSPort,
		Handler: router,
	}

	eg, egCtx := errgroup.WithContext(ctx)

	// 启动 WebSocket 服务
	eg.Go(func() error {
		log.Printf("WebSocket server starting on :%s", cfg.WSPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("ws server listen failed: %w", err)
		}
		return nil
	})

	// 优雅停机逻辑
	eg.Go(func() error {
		<-egCtx.Done()
		log.Println("Received shutdown signal, initiating graceful shutdown...")

		// 关键优化：先切断长连接并通知客户端
		log.Println("Broadcasting shutdown message and closing WebSocket connections...")
		hub.BroadcastSystemAll("服务器即将重启维护，请稍后重新连接...")
		time.Sleep(100 * time.Millisecond)
		hub.Close()

		// 然后再关闭底层的 HTTP Server
		log.Println("Shutting down underlying HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("HTTP server forced to shutdown: %w", err)
		}

		return nil
	})

	if err := eg.Wait(); err != nil {
		log.Printf("WebSocket server error: %v", err)
	}

	log.Println("WebSocket server successfully exited")
}

func setupWSRouter(cfg *config.Config, hub *ws.Hub) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	router.GET("/ws", func(c *gin.Context) {
		validateToken := func(token string) (string, string, error) {
			claims, err := jwt.ValidateToken(token, cfg.JWTSecret)
			if err != nil {
				return "", "", err
			}
			return claims.UserID, claims.Username, nil
		}
		ws.ServeWS(hub, cfg.WSAllowedOrigin, cfg.WSReadLimit, validateToken, c.Writer, c.Request)
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "ws", "timestamp": time.Now().Unix()})
	})

	return router
}
