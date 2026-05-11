.PHONY: all build run start test bench clean proto

all: build

build:
	go build -o bin/api ./cmd/api/main.go
	go build -o bin/grpc ./cmd/grpc/server.go
	go build -o bin/ws ./cmd/ws/server.go

run:
	go run ./cmd/api/main.go

start:
	@./start.bat

run-grpc:
	go run ./cmd/grpc/server.go

run-ws:
	go run ./cmd/ws/server.go

test:
	go test ./... -v

bench:
	go run ./tools/benchmark/benchmark.go

hotreload:
	go run ./tools/hotreload/hotreload.go

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/service.proto

clean:
	rm -rf bin/
	rm -f proto/*.pb.go
