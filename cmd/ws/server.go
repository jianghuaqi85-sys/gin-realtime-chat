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

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"

	"github.com/example/gin-high-performance/internal/config"
	"github.com/example/gin-high-performance/internal/logger"
	"github.com/example/gin-high-performance/pkg/jwt"
	"github.com/example/gin-high-performance/pkg/ws"
)

func main() {
	log.Println("Starting WebSocket server...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger.Init(cfg.LogLevel)

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	hub := ws.NewHub()

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

	srv := &http.Server{
		Addr:    ":" + cfg.WSPort,
		Handler: router,
	}

	eg, ctx := errgroup.WithContext(context.Background())

	eg.Go(func() error {
		log.Printf("WebSocket server starting on :%s", cfg.WSPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("ws server listen failed: %w", err)
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

		log.Println("Shutting down WebSocket server...")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("ws server shutdown failed: %w", err)
		}
		log.Println("Closing WebSocket hub...")
		hub.Close()
		return nil
	})

	if err := eg.Wait(); err != nil {
		log.Fatalf("WebSocket server error: %v", err)
	}

	log.Println("WebSocket server exiting")
}
