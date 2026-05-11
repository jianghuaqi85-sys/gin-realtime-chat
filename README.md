# Gin High Performance Web Framework

A high-performance Gin-based web service with WebSocket, gRPC, JWT, and OpenTelemetry integration.

## Features

- **High Performance**: Optimized for QPS > 12,000 with sub-45ms response latency
- **WebSocket Support**: Real-time bidirectional communication
- **gRPC Integration**: High-performance RPC communication
- **JWT Authentication**: Secure token-based authentication
- **OpenTelemetry**: Distributed tracing and monitoring
- **Rate Limiting**: Redis-based rate limiting
- **Hot Reload**: Development efficiency improvement
- **HTTP/2 Support**: Enhanced protocol performance

## Tech Stack

- Go 1.22+
- Gin Framework
- WebSocket (gorilla/websocket)
- gRPC
- OpenTelemetry
- JWT (golang-jwt)
- Redis
- Jaeger

## Quick Start

### Prerequisites

- Go 1.22+
- Redis (for rate limiting)
- Jaeger (optional, for tracing)

### Installation

```bash
go mod download
```

### Running

```bash
# Run API server
make run

# Run gRPC server
make run-grpc

# Run WebSocket server
make run-ws

# Hot reload (development)
make hotreload
```

### Building

```bash
make build
```

### Benchmark

```bash
make bench
```

## API Endpoints

### Public

- `POST /api/public/login` - Login
- `GET /api/public/health` - Health check

### Protected

- `GET /api/me` - Get current user
- `POST /api/echo` - Echo request body
- `GET /api/ws` - WebSocket connection

## Configuration

Create `.env` file:

```env
APP_ENV=development
APP_PORT=8080
GRPC_PORT=9090
WS_PORT=8081

JWT_SECRET=your-256-bit-secret-key-here
JWT_EXPIRE_HOURS=24

REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

JAEGER_ENDPOINT=http://localhost:14268/api/traces
```

## Project Structure

```
.
├── cmd/
│   ├── api/           # HTTP API server
│   ├── grpc/          # gRPC server
│   └── ws/            # WebSocket server
├── internal/
│   ├── config/        # Configuration
│   ├── handler/       # HTTP handlers
│   ├── logger/        # Logging
│   ├── middleware/    # Middleware
│   ├── repository/    # Data access
│   └── service/       # Business logic
├── pkg/
│   ├── jwt/           # JWT utilities
│   ├── limiter/       # Rate limiter
│   ├── otel/          # OpenTelemetry
│   └── ws/            # WebSocket utilities
├── proto/             # gRPC protocol definitions
└── tools/
    ├── benchmark/     # Performance testing
    └── hotreload/     # Hot reload utility
```
