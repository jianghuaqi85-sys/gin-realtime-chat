package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/example/gin-high-performance/internal/config"
	"github.com/example/gin-high-performance/internal/service"
	pb "github.com/example/gin-high-performance/proto"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge: 5 * time.Minute,
		}),
		grpc.ChainUnaryInterceptor(
			loggingInterceptor,
			recoveryInterceptor,
		),
	)

	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	greeterSvc := service.NewGreeterService()
	pb.RegisterGreeterServer(s, greeterSvc)

	eg, ctx := errgroup.WithContext(context.Background())

	eg.Go(func() error {
		log.Printf("gRPC server listening on :%s", cfg.GRPCPort)
		if err := s.Serve(lis); err != nil {
			return fmt.Errorf("gRPC serve failed: %w", err)
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

		healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		done := make(chan struct{})
		go func() {
			s.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
			log.Println("gRPC server stopped gracefully")
		case <-shutdownCtx.Done():
			log.Println("gRPC shutdown deadline exceeded, forcing stop")
			s.Stop()
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}

	log.Println("gRPC server exiting")
}

func loggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	log.Printf("gRPC %s %v err=%v", info.FullMethod, time.Since(start), err)
	return resp, err
}

func recoveryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("gRPC panic recovered in %s: %v", info.FullMethod, r)
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()
	return handler(ctx, req)
}
