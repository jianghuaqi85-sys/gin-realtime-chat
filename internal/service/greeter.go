package service

import (
	"context"

	pb "github.com/example/gin-high-performance/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type GreeterService struct {
	pb.UnimplementedGreeterServer
}

func NewGreeterService() *GreeterService {
	return &GreeterService{}
}

func (s *GreeterService) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	// 1. 上下文取消/超时探活
	if err := ctx.Err(); err != nil {
		return nil, status.Error(codes.Canceled, "request canceled")
	}

	name := req.GetName()

	// 2. 参数基础校验，返回标准 gRPC 状态码
	if len(name) > 100 {
		return nil, status.Error(codes.InvalidArgument, "name exceeds maximum length")
	}

	if name == "" {
		name = "World"
	}

	// 3. 高性能字符串拼接（免除 fmt.Sprintf 反射开销）
	return &pb.HelloResponse{
		Message: "Hello, " + name + "!",
	}, nil
}
