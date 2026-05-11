package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
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

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger.Init(cfg.LogLevel)

	shutdownTracer, err := otel.InitTracer(cfg.OtelEndpoint, cfg.OtelInsecure)
	if err != nil {
		log.Printf("OTel tracer init failed, tracing disabled: %v", err)
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := shutdownTracer(shutdownCtx); err != nil {
				log.Printf("OTel tracer shutdown error: %v", err)
			}
		}()
	}

	var rateLimiter middleware.RateLimiter
	var messageBus *redisbus.MessageBus

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	_, err = rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Printf("Redis connection failed, rate limiting disabled: %v", err)
		rateLimiter = nil
	} else {
		log.Printf("Redis connected successfully")
		rateLimiter = limiter.NewLimiter(rdb)
		messageBus = redisbus.NewMessageBus(rdb)
		log.Println("Redis Pub/Sub message bus enabled")
	}

	// 启动 Cloudflare Tunnel（如果配置了 cloudflared 路径）
	if cfg.CloudflaredPath != "" {
		tm := tunnel.NewManager(cfg.CloudflaredPath, cfg.AppPort, ".tunnel_url")
		if err := tm.Start(); err != nil {
			log.Printf("Cloudflare Tunnel 启动失败: %v", err)
		} else {
			log.Println("Cloudflare Tunnel 正在启动...")
			defer tm.Stop()
		}
	}

	var userRepo repository.UserRepository
	var channelRepo repository.ChannelRepository
	var messageRepo repository.MessageRepository

	if cfg.MySQLDSN != "" {
		db, err := database.Connect(cfg.MySQLDSN, cfg.DBLogLevel)
		if err != nil {
			log.Fatalf("Failed to connect to MySQL: %v", err)
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

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(gzip.Gzip(gzip.DefaultCompression))
	router.Use(middleware.OtelMiddleware())
	router.Use(middleware.LoggingMiddleware())

	authHandler := handler.NewAuthHandler(cfg, userRepo)
	hub := ws.NewHub()
	if messageBus != nil {
		hub.SetBus(messageBus)
	}

	// 聊天功能（需要 MySQL）
	if channelRepo != nil && messageRepo != nil {
		chatHandler := handler.NewChatHandler(channelRepo, messageRepo, hub, messageBus)
		hub.OnMessage = chatHandler.OnWSMessage

		router.GET("/", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.String(http.StatusOK, chatHTML)
		})

		api := router.Group("/api")
		api.Use(middleware.AuthMiddleware(cfg))
		api.Use(middleware.RateLimitMiddleware(rateLimiter, 100, time.Minute))
		api.GET("/me", authHandler.Me)
		api.PUT("/password", authHandler.ChangePassword)
		api.POST("/channels", chatHandler.CreateChannel)
		api.GET("/channels", chatHandler.ListChannels)
		api.GET("/channels/:id/messages", chatHandler.GetMessages)
		api.PUT("/messages/:id", chatHandler.EditMessage)
		api.DELETE("/messages/:id", chatHandler.DeleteMyMessage)
		api.GET("/online", func(c *gin.Context) {
			c.JSON(http.StatusOK, hub.OnlineUsers())
		})

		// 管理员接口
		adminHandler := handler.NewAdminHandler(userRepo, channelRepo, messageRepo, hub)
		api.GET("/tunnel", adminHandler.Tunnel)
		admin := api.Group("/admin")
		admin.Use(middleware.AdminMiddleware(userRepo))
		admin.GET("/stats", adminHandler.Stats)
		admin.GET("/users", adminHandler.ListUsers)
		admin.DELETE("/users/:id", adminHandler.DeleteUser)
		admin.POST("/ban", adminHandler.Ban)
		admin.POST("/unban", adminHandler.Unban)
		admin.DELETE("/channels/:id", adminHandler.DeleteChannel)
		admin.DELETE("/channels/:id/messages", adminHandler.ClearMessages)
		admin.DELETE("/messages/:id", adminHandler.DeleteMessage)
		admin.POST("/broadcast", adminHandler.Broadcast)

		// WebSocket — token 通过第一条 auth 消息传递，避免 URL 泄露
		router.GET("/api/ws", func(c *gin.Context) {
			validateToken := func(token string) (string, string, error) {
				claims, err := jwt.ValidateToken(token, cfg.JWTSecret)
				if err != nil {
					return "", "", err
				}
				// 检查用户是否被封禁
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

	public := router.Group("/api/public")
	public.Use(middleware.RateLimitMiddleware(rateLimiter, 20, time.Minute))
	public.POST("/login", authHandler.Login)
	public.POST("/register", authHandler.Register)
	public.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "timestamp": time.Now().Unix()})
	})

	srv := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	eg, ctx := errgroup.WithContext(context.Background())

	eg.Go(func() error {
		log.Printf("API server starting on :%s", cfg.AppPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server listen failed: %w", err)
		}
		return nil
	})

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	eg.Go(func() error {
		select {
		case <-quit:
			log.Println("Received shutdown signal")
		case <-ctx.Done():
			log.Println("Context cancelled")
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Println("Shutting down API server...")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
		log.Println("Notifying WebSocket clients before shutdown...")
		hub.BroadcastSystemAll("服务器即将关闭，请稍后重新连接")
		time.Sleep(100 * time.Millisecond)
		log.Println("Closing WebSocket hub...")
		hub.Close()
		return nil
	})

	if err := eg.Wait(); err != nil {
		log.Fatalf("Server error: %v", err)
	}

	log.Println("Server exiting")
}
