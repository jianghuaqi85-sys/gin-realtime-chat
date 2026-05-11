package service

import (
	"context"
	"fmt"

	pb "github.com/example/gin-high-performance/proto"
)

type GreeterService struct {
	pb.UnimplementedGreeterServer
}

func NewGreeterService() *GreeterService {
	return &GreeterService{}
}

func (s *GreeterService) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	name := req.GetName()
	if name == "" {
		name = "World"
	}
	return &pb.HelloResponse{
		Message: fmt.Sprintf("Hello, %s!", name),
	}, nil
}
