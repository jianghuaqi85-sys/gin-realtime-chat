package main

import (
	"context"
	"fmt"
	"log"
	"net"
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

	// 1. 使用更现代的上下文监听系统信号 (Go 1.16+)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 2. 配置 gRPC Server
	s := grpc.NewServer(
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionAge: 5 * time.Minute,
		}),
		grpc.ChainUnaryInterceptor(
			loggingInterceptor,
			recoveryInterceptor,
		),
	)

	// 3. 注册健康检查
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	// 4. 注册业务服务
	greeterSvc := service.NewGreeterService()
	pb.RegisterGreeterServer(s, greeterSvc)

	// 5. 使用 errgroup 管理生命周期
	eg, egCtx := errgroup.WithContext(ctx)

	// 启动 gRPC 服务
	eg.Go(func() error {
		log.Printf("gRPC server listening on :%s", cfg.GRPCPort)
		if err := s.Serve(lis); err != nil {
			return fmt.Errorf("gRPC serve failed: %w", err)
		}
		return nil
	})

	// 监听停机信号
	eg.Go(func() error {
		<-egCtx.Done()
		log.Println("Received shutdown signal, initiating graceful shutdown...")

		// (关键优化): 标记不健康，并休眠等待 LB 摘除节点 (K8s 黄金法则)
		healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
		time.Sleep(2 * time.Second)

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
		log.Printf("gRPC server exit context: %v", err)
	}

	log.Println("gRPC server exiting")
}

func loggingInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	if err != nil {
		st, _ := status.FromError(err)
		log.Printf("[gRPC Error] %s | %v | code: %s | err: %v", info.FullMethod, time.Since(start), st.Code().String(), err)
	} else {
		log.Printf("[gRPC Info] %s | %v", info.FullMethod, time.Since(start))
	}
	return resp, err
}

func recoveryInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[gRPC Panic] recovered in %s: %v", info.FullMethod, r)
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()
	return handler(ctx, req)
}
