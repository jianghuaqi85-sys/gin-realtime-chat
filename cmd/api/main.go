package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"golang.org/x/sync/errgroup"

	"github.com/example/gin-high-performance/internal/config"
	"github.com/example/gin-high-performance/internal/database"
	"github.com/example/gin-high-performance/internal/handler"
	"github.com/example/gin-high-performance/internal/logger"
	"github.com/example/gin-high-performance/internal/middleware"
	"github.com/example/gin-high-performance/internal/repository"
	"github.com/example/gin-high-performance/internal/tunnel"
	"github.com/example/gin-high-performance/pkg/jwt"
	"github.com/example/gin-high-performance/pkg/limiter"
	"github.com/example/gin-high-performance/pkg/otel"
	"github.com/example/gin-high-performance/pkg/redisbus"
	"github.com/example/gin-high-performance/pkg/ws"
)

func main() {
	log.Println("Starting API server...")

	// 1. 初始化配置与日志
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	logger.Init(cfg.LogLevel)

	// 2. 使用 Signal Context 监听系统信号 (SIGINT, SIGTERM)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 3. 初始化 OTel Tracing
	shutdownTracer, err := otel.InitTracer(ctx, cfg.OtelEndpoint, cfg.OtelInsecure)
	if err != nil {
		log.Printf("OTel tracer init failed, tracing disabled: %v", err)
	} else {
		defer func() {
			tCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := shutdownTracer(tCtx); err != nil {
				log.Printf("OTel tracer shutdown error: %v", err)
			}
		}()
	}

	// 4. 初始化 Redis 客户端与连接回收
	var rateLimiter middleware.RateLimiter
	var messageBus *redisbus.MessageBus
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Printf("Error closing Redis connection: %v", err)
		}
	}()

	if _, err = rdb.Ping(context.Background()).Result(); err != nil {
		log.Printf("Redis connection failed, rate limiting disabled: %v", err)
	} else {
		log.Printf("Redis connected successfully")
		rateLimiter = limiter.NewLimiter(rdb)
		messageBus = redisbus.NewMessageBus(rdb)
		defer func() {
			if err := messageBus.Close(); err != nil {
				log.Printf("Error closing message bus: %v", err)
			}
		}()
		log.Println("Redis Pub/Sub message bus enabled")
	}

	// 5. 启动 Cloudflare Tunnel
	if cfg.CloudflaredPath != "" {
		tm := tunnel.NewManager(cfg.CloudflaredPath, cfg.AppPort, ".tunnel_url")
		tunnelCtx, tunnelCancel := context.WithTimeout(ctx, 15*time.Second)
		defer tunnelCancel()
		if err := tm.Start(tunnelCtx); err != nil {
			log.Printf("Cloudflare Tunnel 启动失败: %v", err)
		} else {
			log.Printf("Cloudflare Tunnel 地址就绪: %s", tm.GetURL())
			defer tm.Stop()
		}
	}

	// 6. 初始化 MySQL 数据库与连接池回收
	var userRepo repository.UserRepository
	var channelRepo repository.ChannelRepository
	var messageRepo repository.MessageRepository

	if cfg.MySQLDSN != "" {
		db, err := database.Connect(cfg.MySQLDSN, cfg.DBLogLevel)
		if err != nil {
			log.Fatalf("Failed to connect to MySQL: %v", err)
		}
		if sqlDB, err := db.DB(); err == nil {
			defer sqlDB.Close()
		}
		if err := database.AutoMigrate(db, &repository.User{}, &repository.Channel{}, &repository.Message{}); err != nil {
			log.Fatalf("Failed to auto-migrate MySQL tables: %v", err)
		}
		userRepo = repository.NewMySQLUserRepository(db)
		channelRepo = repository.NewMySQLChannelRepository(db)
		messageRepo = repository.NewMySQLMessageRepository(db)
		log.Println("Using MySQL repositories")
	} else {
		userRepo = repository.NewInMemoryUserRepository()
		log.Println("Using in-memory user repository (MYSQL_DSN not set)")
	}

	// 7. 初始化 WebSocket Hub
	hub := ws.NewHub()
	if messageBus != nil {
		hub.SetBus(messageBus)
	}

	// 8. 路由与中间件装配
	router := setupRouter(cfg, userRepo, channelRepo, messageRepo, hub, messageBus, rateLimiter)

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	// 9. 启动 HTTP 服务与优雅停机
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		log.Printf("API server starting on :%s", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server listen failed: %w", err)
		}
		return nil
	})

	eg.Go(func() error {
		<-egCtx.Done()
		log.Println("Initiating graceful shutdown...")

		// 优化停序：优先广播并断开 WebSocket 长连接
		log.Println("Notifying and disconnecting WebSocket clients...")
		hub.BroadcastSystemAll("服务器即将关闭维护，请稍后重新连接...")
		time.Sleep(100 * time.Millisecond)
		hub.Close()

		// 再关闭 HTTP 服务
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Println("Shutting down HTTP server...")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("HTTP server forced shutdown: %w", err)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		log.Printf("Server exit context: %v", err)
	}
	log.Println("Server successfully stopped")
}

// setupRouter 抽离路由注册逻辑
func setupRouter(
	cfg *config.Config,
	userRepo repository.UserRepository,
	channelRepo repository.ChannelRepository,
	messageRepo repository.MessageRepository,
	hub *ws.Hub,
	messageBus *redisbus.MessageBus,
	rateLimiter middleware.RateLimiter,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.OtelMiddleware())
	router.Use(middleware.LoggingMiddleware(cfg))

	authHandler := handler.NewAuthHandler(cfg, userRepo, hub)

	// 公共接口
	public := router.Group("/api/public")
	public.Use(middleware.RateLimitMiddleware(rateLimiter, 20, time.Minute))
	public.POST("/login", authHandler.Login)
	public.POST("/register", authHandler.Register)
	public.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().Unix()})
	})

	// 聊天功能 (依赖 MySQL)
	if channelRepo != nil && messageRepo != nil {
		chatHandler := handler.NewChatHandler(channelRepo, messageRepo, hub, messageBus)
		hub.OnMessage = chatHandler.OnWSMessage

		router.GET("/", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, chatHTML)
		})

		api := router.Group("/api")
		api.Use(middleware.AuthMiddleware(cfg, userRepo))
		api.Use(middleware.RateLimitMiddleware(rateLimiter, 100, time.Minute))
		api.GET("/me", authHandler.Me)
		api.PUT("/password", authHandler.ChangePassword)
		api.PUT("/username", authHandler.ChangeUsername)
		api.GET("/channels", chatHandler.ListChannels)
		api.GET("/channels/:id/messages", chatHandler.GetMessages)
		api.PUT("/messages/:id", chatHandler.EditMessage)
		api.DELETE("/messages/:id", chatHandler.DeleteMyMessage)
		api.GET("/online", func(c *gin.Context) {
			c.JSON(http.StatusOK, hub.OnlineUsers())
		})

		// 管理员接口
		adminHandler := handler.NewAdminHandler(userRepo, channelRepo, messageRepo, hub)
		admin := api.Group("/admin")
		admin.Use(middleware.AdminMiddleware(userRepo))
		admin.GET("/tunnel", adminHandler.Tunnel)
		admin.GET("/stats", adminHandler.Stats)
		admin.GET("/users", adminHandler.ListUsers)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)
		admin.POST("/ban", adminHandler.Ban)
		admin.POST("/unban", adminHandler.Unban)
		admin.POST("/channels", chatHandler.CreateChannel)
		admin.DELETE("/channels/:id", adminHandler.DeleteChannel)
		admin.DELETE("/channels/:id/messages", adminHandler.ClearMessages)
		admin.DELETE("/messages/:id", adminHandler.DeleteMessage)
		admin.POST("/broadcast", adminHandler.Broadcast)

		// 超级管理员接口
		superAdmin := api.Group("/admin")
		superAdmin.Use(middleware.SuperAdminMiddleware(userRepo))
		superAdmin.POST("/set-admin", adminHandler.SetAdmin)
		superAdmin.POST("/remove-admin", adminHandler.RemoveAdmin)

		// WebSocket 入口
		router.GET("/api/ws", func(c *gin.Context) {
			validateToken := func(token string) (string, string, error) {
				claims, err := jwt.ValidateToken(token, cfg.JWTSecret)
				if err != nil {
					return "", "", err
				}
				user, err := userRepo.GetUserByUsername(claims.Username)
				if err != nil || user == nil {
					return "", "", fmt.Errorf("user not found")
				}
				if user.Banned {
					return "", "", fmt.Errorf("account banned")
				}
				return claims.UserID, claims.Username, nil
			}
			ws.ServeWS(hub, cfg.WSAllowedOrigin, cfg.WSReadLimit, validateToken, c.Writer, c.Request)
		})
	} else {
		router.GET("/", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, `<html><body><h1>Gin High Performance</h1><p>聊天功能需要 MySQL。请配置 MYSQL_DSN。</p></body></html>`)
		})
	}

	return router
}
