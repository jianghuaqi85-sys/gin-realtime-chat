# GIn -- Gin High Performance Chat Server

A high-performance real-time chat server built with Go and the Gin framework, featuring WebSocket-based messaging, multi-channel support, admin management panel, Redis Pub/Sub for horizontal scaling, Cloudflare Tunnel for instant public access, and a built-in single-page chat UI.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Language | Go 1.25 |
| HTTP Framework | Gin v1.10 |
| Database | MySQL 8.x (via GORM v1.31) |
| Cache / Message Bus | Redis (go-redis v8) |
| WebSocket | Gorilla WebSocket v1.5 |
| Authentication | JWT (golang-jwt v5) + bcrypt |
| Configuration | Viper (`.env` file + environment variables) |
| Logging | Logrus (structured JSON) |
| Tracing | OpenTelemetry (OTLP HTTP / stdout) |
| gRPC | google.golang.org/grpc v1.80 + Protobuf |
| Compression | gin-contrib/gzip |
| Rate Limiting | Redis sliding-window (Lua script) |
| Tunnel | Cloudflare Tunnel (`cloudflared`) |
| Frontend | Single-page HTML/CSS/JS (embedded in Go binary) |

---

## Features

### Core Chat

- User registration and login with bcrypt password hashing
- JWT-based authentication (HS256, configurable expiry)
- Multi-channel chat: create, list, and switch between channels
- Real-time messaging via WebSocket with automatic reconnection
- Message editing and deletion (own messages only)
- Message history with cursor-based pagination (`before` timestamp)
- System messages for user join/leave events
- Online user list with periodic refresh
- Password change (requires old password verification)
- Session persistence via localStorage (auto-login on reload)

### Admin Panel

- Server statistics dashboard (online users, total users, channels, messages)
- User management: list, delete, ban, unban
- Channel management: list, delete, clear messages
- Global broadcast announcements to all connected users
- Admin-only route protection with role-based middleware

### Real-Time Infrastructure

- WebSocket Hub with 256 hash-bucketed client pools for lock contention reduction
- 64-shard channel locks for fine-grained concurrency
- Write-behind message persistence: broadcast first, async persist to DB
- Redis Pub/Sub message bus for horizontal multi-instance scaling
- Graceful shutdown with client notification before disconnect
- Exponential backoff reconnection with fallback to HTTP polling (client-side)
- Origin checking for WebSocket upgrade requests

### Performance Optimizations

- Gzip compression middleware on all HTTP responses
- Connection pool tuning: 50 idle, 200 max open, 1h lifetime
- Sliding-window rate limiter via atomic Redis Lua script
- sync.Pool for logging field maps to reduce GC pressure
- Persistent worker pool (NumCPU goroutines) for async DB writes
- FNV-32a hash-based sharding for both client buckets and channel locks
- Read/write deadline management on WebSocket connections

### Observability

- OpenTelemetry distributed tracing (OTLP HTTP export or stdout)
- Structured JSON logging via Logrus with request metadata
- Health check endpoint (`/api/public/health`)

### Deployment

- Cloudflare Tunnel integration (auto-start, URL capture, public access)
- ngrok tunnel URL detection via local API
- One-click `start.bat` for Windows (Redis + API + Tunnel)
- Built-in hot-reload development tool
- Built-in benchmark tool for load testing

---

## Project Structure

```
GIn/
├── cmd/
│   ├── api/
│   │   ├── main.go            # API server entry point, routing, graceful shutdown
│   │   └── chat_html.go       # Embedded single-page chat UI (HTML/CSS/JS)
│   ├── grpc/
│   │   └── server.go          # Standalone gRPC server (Greeter service + health)
│   └── ws/
│       └── server.go          # Standalone WebSocket server
├── internal/
│   ├── config/
│   │   └── config.go          # Viper-based configuration loader with validation
│   ├── database/
│   │   └── database.go        # GORM MySQL connection with pool tuning
│   ├── handler/
│   │   ├── auth_handler.go    # Login, register, me, change password
│   │   ├── chat_handler.go    # Channels, messages, WS message handler
│   │   └── admin_handler.go   # Admin stats, user/channel/message management, broadcast
│   ├── logger/
│   │   └── logger.go          # Logrus initialization and convenience wrappers
│   ├── middleware/
│   │   ├── auth.go            # JWT Bearer token authentication
│   │   ├── admin.go           # Admin role verification (with DB fallback)
│   │   ├── rate_limit.go      # Redis sliding-window rate limiter middleware
│   │   ├── logging.go         # Request logging with sync.Pool optimization
│   │   └── otel.go            # OpenTelemetry span creation middleware
│   ├── repository/
│   │   ├── user_repository.go # User model + InMemory / MySQL implementations
│   │   └── chat_repository.go # Channel + Message models + MySQL implementations
│   ├── service/
│   │   └── greeter.go         # gRPC Greeter service implementation
│   └── tunnel/
│       └── tunnel.go          # Cloudflare Tunnel process manager
├── pkg/
│   ├── jwt/
│   │   └── jwt.go             # JWT generation, parsing, validation (HS256)
│   ├── limiter/
│   │   └── limiter.go         # Redis Lua sliding-window rate limiter
│   ├── otel/
│   │   └── otel.go            # OpenTelemetry tracer provider initialization
│   ├── redisbus/
│   │   └── bus.go             # Redis Pub/Sub message bus for cross-instance broadcast
│   └── ws/
│       └── ws.go              # WebSocket Hub, Bucket, Client, channel sharding
├── proto/
│   └── service.proto          # Protobuf definition for Greeter service
├── tools/
│   ├── benchmark/
│   │   └── benchmark.go       # HTTP load testing tool (QPS, latency, P99)
│   └── hotreload/
│       └── hotreload.go       # File watcher for automatic server restart
├── .env                       # Environment configuration (not committed)
├── go.mod                     # Go module definition
├── go.sum                     # Dependency checksums
├── Makefile                   # Build, run, test, proto generation commands
└── start.bat                  # Windows one-click startup (Redis + API + Tunnel)
```

---

## Quick Start

### Prerequisites

- Go 1.25+
- Redis 6.x+
- MySQL 8.x+ (optional; without it, only in-memory user storage is available, chat features are disabled)
- `cloudflared` CLI (optional, for public tunnel)

### 1. Clone and Configure

```bash
git clone https://github.com/example/gin-high-performance.git
cd gin-high-performance
cp .env.example .env   # or create .env manually
```

Edit `.env` with your settings (minimum required: `JWT_SECRET` with at least 32 characters).

### 2. Install Dependencies

```bash
go mod download
```

### 3. Run

**Option A: Make**

```bash
make run          # Run API server directly
make build        # Build all binaries to bin/
make start        # Run start.bat (Windows)
```

**Option B: Go directly**

```bash
go run ./cmd/api/main.go
```

**Option C: Windows one-click**

Double-click `start.bat` -- it automatically starts Redis, the API server, and Cloudflare Tunnel.

### 4. Access

- Local: `http://localhost:8080`
- Public: check the Cloudflare Tunnel window or the tunnel URL in the UI sidebar

### 5. Default Admin Account

| Field | Value |
|---|---|
| Username | `admin` |
| Password | `admin123` |

Change this password immediately after first login.

---

## API Documentation

Base URL: `http://localhost:8080`

All authenticated endpoints require the `Authorization: Bearer <token>` header.

### Public Endpoints (no auth required)

| Method | Path | Description | Rate Limit |
|---|---|---|---|
| `POST` | `/api/public/register` | Register a new user | 20/min per IP |
| `POST` | `/api/public/login` | Login, returns JWT token | 20/min per IP |
| `GET` | `/api/public/health` | Health check | 20/min per IP |

#### POST /api/public/register

```json
// Request
{ "username": "alice", "password": "secret123", "confirm_password": "secret123" }

// Success 201
{ "message": "user created" }

// Error 409
{ "error": "用户名已被占用" }
```

Validation rules:
- Username: 3-32 characters, alphanumeric + underscore only
- Password: minimum 8 characters
- Confirm password must match

#### POST /api/public/login

```json
// Request
{ "username": "alice", "password": "secret123" }

// Success 200
{ "token": "eyJhbGciOiJIUzI1NiIs..." }

// Error 401
{ "error": "用户名或密码错误" }
```

#### GET /api/public/health

```json
// Success 200
{ "status": "ok", "timestamp": 1715000000 }
```

### Authenticated Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/me` | Get current user info |
| `PUT` | `/api/password` | Change password |
| `GET` | `/api/channels` | List all channels |
| `POST` | `/api/channels` | Create a new channel |
| `GET` | `/api/channels/:id/messages` | Get channel messages (paginated) |
| `PUT` | `/api/messages/:id` | Edit own message |
| `DELETE` | `/api/messages/:id` | Delete own message |
| `GET` | `/api/online` | List online users |
| `GET` | `/api/tunnel` | Get public tunnel URLs |

Rate limit for authenticated endpoints: 100 requests/minute per IP.

#### GET /api/me

```json
// Success 200
{ "user_id": "uuid...", "username": "alice" }
```

#### PUT /api/password

```json
// Request
{ "old_password": "secret123", "new_password": "newsecret456" }

// Success 200
{ "message": "密码修改成功" }
```

#### POST /api/channels

```json
// Request
{ "name": "general" }

// Success 201
{ "ID": "uuid...", "Name": "general", "CreatedBy": "uuid...", "CreatedAt": "..." }
```

#### GET /api/channels/:id/messages

Query parameters:
- `limit` (default: 50, max: 200) -- number of messages
- `before` (optional, RFC3339 timestamp) -- cursor for pagination

```json
// Success 200
[
  {
    "ID": "uuid...",
    "ChannelID": "uuid...",
    "UserID": "uuid...",
    "Username": "alice",
    "Content": "Hello!",
    "CreatedAt": "2026-05-12T10:00:00Z"
  }
]
```

#### PUT /api/messages/:id

```json
// Request
{ "content": "Updated message text" }

// Success 200
{ "message": "编辑成功" }
```

#### DELETE /api/messages/:id

```json
// Success 200
{ "message": "删除成功" }

// Error 403
{ "error": "只能删除自己的消息" }
```

#### GET /api/tunnel

```json
// Success 200
{ "urls": ["https://random-name.trycloudflare.com"] }

// Error 404
{ "error": "没有检测到隧道，请确认 ngrok 或 Cloudflare Tunnel 已启动" }
```

### Admin Endpoints

All admin endpoints require authentication + admin role.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/admin/stats` | Server statistics |
| `GET` | `/api/admin/users` | List all users |
| `DELETE` | `/api/admin/users/:id` | Delete a user (disconnects first) |
| `POST` | `/api/admin/ban` | Ban a user |
| `POST` | `/api/admin/unban` | Unban a user |
| `DELETE` | `/api/admin/channels/:id` | Delete channel and its messages |
| `DELETE` | `/api/admin/channels/:id/messages` | Clear all messages in a channel |
| `DELETE` | `/api/admin/messages/:id` | Delete any message |
| `POST` | `/api/admin/broadcast` | Send system announcement to all users |

#### GET /api/admin/stats

```json
{
  "users_total": 42,
  "online": 7,
  "channels": 5,
  "messages": 1234
}
```

#### GET /api/admin/users

```json
[
  { "id": "uuid...", "username": "admin", "role": "admin", "banned": false },
  { "id": "uuid...", "username": "alice", "role": "user", "banned": false }
]
```

#### POST /api/admin/ban

```json
// Request
{ "user_id": "uuid..." }

// Success 200
{ "banned": "uuid..." }
```

#### POST /api/admin/unban

```json
// Request
{ "user_id": "uuid..." }

// Success 200
{ "unbanned": "uuid..." }
```

#### POST /api/admin/broadcast

```json
// Request
{ "content": "系统维护通知：今晚 22:00 将进行升级" }

// Success 200
{ "broadcast": "系统维护通知：今晚 22:00 将进行升级" }
```

---

## WebSocket Protocol

Endpoint: `ws://localhost:8080/api/ws` (or `wss://` over TLS)

### Connection Flow

```
Client                          Server
  |                               |
  |--- WS Upgrade Request ------->|
  |<-- 101 Switching Protocols ---|
  |                               |
  |--- { type: "auth", token } -->|   (must be first message, 10s timeout)
  |<-- { type: "auth_ok" } -------|
  |                               |
  |--- { type: "join", channel_id } -->|
  |<-- { type: "system", content: "alice joined" } --|
  |                               |
  |--- { type: "message", channel_id, content } -->|
  |<-- { type: "message", ... } --|  (broadcast to all channel members)
  |                               |
  |--- { type: "leave", channel_id } -->|
  |<-- { type: "system", content: "alice left" } --|
```

### Message Types

#### Client -> Server

| type | Fields | Description |
|---|---|---|
| `auth` | `token` | Authenticate with JWT. Must be first message. |
| `join` | `channel_id` | Join a channel to receive its messages |
| `leave` | `channel_id` | Leave a channel |
| `message` | `channel_id`, `content` | Send a message to a joined channel |

#### Server -> Client

| type | Fields | Description |
|---|---|---|
| `auth_ok` | `content` (username) | Authentication successful |
| `error` | `content` | Error message (auth failure, not in channel, etc.) |
| `message` | `channel_id`, `user_id`, `username`, `content`, `created_at` | New message in a channel |
| `system` | `channel_id` (optional), `content`, `created_at` | System notification (join/leave/broadcast) |

### Connection Parameters

| Parameter | Value |
|---|---|
| Ping interval | 54 seconds |
| Pong timeout | 60 seconds |
| Write deadline | 10 seconds |
| Auth timeout | 10 seconds |
| Default read limit | 512 bytes (configurable via `WS_READ_LIMIT`) |
| Send buffer per client | 256 messages |

### Client-Side Reconnection Strategy

The built-in UI implements:
1. Exponential backoff: 1s -> 2s -> 4s -> 8s -> 16s -> 30s (cap)
2. After 5 consecutive failures, falls back to HTTP polling (every 5 seconds)
3. Automatically resumes WebSocket when connection is restored

---

## Configuration

All configuration is loaded from `.env` file and/or environment variables. Environment variables take precedence.

| Variable | Type | Default | Description |
|---|---|---|---|
| `APP_ENV` | string | `development` | Application environment (`development` / `production`) |
| `APP_PORT` | string | `8080` | HTTP API server port |
| `GRPC_PORT` | string | `9090` | gRPC server port (standalone mode) |
| `WS_PORT` | string | `8081` | WebSocket server port (standalone mode) |
| `JWT_SECRET` | string | **(required)** | Secret key for JWT signing. Must be at least 32 characters. |
| `JWT_EXPIRE_HOURS` | int | `24` | JWT token expiration time in hours |
| `REDIS_ADDR` | string | `localhost:6379` | Redis server address |
| `REDIS_PASSWORD` | string | *(empty)* | Redis authentication password |
| `REDIS_DB` | int | `0` | Redis database number |
| `MYSQL_DSN` | string | *(empty)* | MySQL connection string. If empty, uses in-memory storage and disables chat features. |
| `OTEL_ENDPOINT` | string | *(empty)* | OpenTelemetry collector endpoint. Set to `stdout` or leave empty for console output. |
| `OTEL_INSECURE` | bool | `true` | Allow insecure OTLP HTTP connection |
| `LOG_LEVEL` | string | `info` | Log level: `trace`, `debug`, `info`, `warn`, `error`, `fatal` |
| `DB_LOG_LEVEL` | string | `warn` | GORM SQL log level: `silent`, `error`, `warn`, `info` |
| `WS_ALLOWED_ORIGIN` | string | *(empty)* | Allowed WebSocket origin. If empty, only same-origin connections are allowed. |
| `WS_READ_LIMIT` | int | `512` | Maximum WebSocket message size in bytes |
| `CLOUDFLARED_PATH` | string | *(empty)* | Path to `cloudflared` executable. If set, auto-starts Cloudflare Tunnel on boot. |

### MySQL DSN Format

```
user:password@tcp(host:port)/dbname?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local
```

---

## Deployment

### Local Development

```bash
# Terminal 1: Start Redis
redis-server

# Terminal 2: Start API server
make run

# Or use hot-reload during development
make hotreload
```

### Build for Production

```bash
make build
# Outputs:
#   bin/api   - HTTP API + WebSocket + embedded UI
#   bin/grpc  - Standalone gRPC server
#   bin/ws    - Standalone WebSocket server
```

### Docker Compose (MySQL + Redis)

Create a `docker-compose.yml` for dependencies:

```yaml
services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: your_password
      MYSQL_DATABASE: gin_high_performance
    ports:
      - "3306:3306"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

### Cloudflare Tunnel

**Automatic (via configuration):**

Set `CLOUDFLARED_PATH` in `.env` to the path of your `cloudflared` executable. The server will automatically start a tunnel on boot and capture the public URL.

```env
CLOUDFLARED_PATH=/usr/local/bin/cloudflared
```

**Manual:**

```bash
cloudflared tunnel --url http://localhost:8080
```

The tunnel URL is displayed in the server logs and in the chat UI sidebar. The URL is also saved to `.tunnel_url` for API access.

**ngrok (alternative):**

If ngrok is running on the default port (4040), the server automatically detects the tunnel URL via the ngrok local API.

### Windows One-Click Start

Edit `start.bat` to configure paths:

```bat
set CLOUDFLARED=D:\cloudflared.exe
set REDIS_SERVER=D:\Redis\redis-server.exe
set REDIS_CONFIG=D:\Redis\redis.windows.conf
```

Then double-click `start.bat`. It will:
1. Start Redis (if not already running)
2. Start the API server
3. Start Cloudflare Tunnel
4. Display local and public access URLs

---

## Architecture

### Request Flow

```
Client Request
    |
    v
[Gin Router]
    |
    +-- Recovery Middleware (panic recovery)
    +-- Gzip Middleware (response compression)
    +-- OTel Middleware (distributed tracing)
    +-- Logging Middleware (structured request logging)
    |
    +-- /api/public/*  --> Rate Limiter (20/min) --> Auth Handler
    +-- /api/*         --> Rate Limiter (100/min) --> Auth Middleware --> Handlers
    +-- /api/admin/*   --> Rate Limiter (100/min) --> Auth Middleware --> Admin Middleware --> Admin Handlers
    +-- /api/ws        --> WebSocket Upgrade --> Auth (first message) --> Hub
```

### WebSocket Architecture

```
                    +------------------+
                    |     Hub          |
                    |  (256 buckets)   |
                    +--------+---------+
                             |
            +----------------+----------------+
            |                |                |
     +------+------+  +-----+------+  +------+------+
     |  Bucket 0   |  |  Bucket 1  |  | Bucket 255  |
     | (goroutine) |  | (goroutine)|  | (goroutine) |
     +------+------+  +-----+------+  +------+------+
            |                |                |
       +----+----+      +----+----+      +----+----+
       | Client  |      | Client  |      | Client  |
       | Client  |      | Client  |      | Client  |
       +---------+      +---------+      +---------+

Channel Shards (64):
  shard[0]: { channelA: [client1, client2], channelB: [client3] }
  shard[1]: { channelC: [client4] }
  ...
  shard[63]: { channelZ: [clientN] }
```

Key design decisions:
- **256 hash-bucketed client pools**: Clients are assigned to buckets by FNV-32a hash of their user ID. Each bucket runs its own goroutine, reducing lock contention compared to a single global lock.
- **64-shard channel locks**: Channel membership is distributed across 64 shards by FNV-32a hash of the channel ID, enabling concurrent channel operations.
- **Write-behind persistence**: Messages are broadcast immediately, then queued to a buffered channel (capacity 4096) for async database writes by a worker pool (NumCPU workers, minimum 2).

### Multi-Instance Scaling (Redis Pub/Sub)

```
  Instance A                    Instance B
  +----------+                  +----------+
  | Hub      |                  | Hub      |
  | Client 1 |                  | Client 3 |
  | Client 2 |                  | Client 4 |
  +----+-----+                  +----+-----+
       |                             |
       v                             v
  +----+-----------------------------+-----+
  |           Redis Pub/Sub                |
  |  Channel: chat:ch:<channel_id>         |
  +----------------------------------------+
```

When `REDIS_ADDR` is configured, the message bus automatically enables Redis Pub/Sub. Messages published to a channel are broadcast to all instances subscribing to that channel. The hub manages subscribe/unsubscribe lifecycle based on local client count.

### Data Models

**User**

| Field | Type | Description |
|---|---|---|
| `id` | varchar(36) | UUID primary key |
| `username` | varchar(64) | Unique, alphanumeric + underscore |
| `password_hash` | varchar(255) | bcrypt hash |
| `role` | varchar(16) | `admin` or `user` |
| `banned` | boolean | Account ban flag |

**Channel**

| Field | Type | Description |
|---|---|---|
| `id` | varchar(36) | UUID primary key |
| `name` | varchar(64) | Unique channel name |
| `created_by` | varchar(36) | Creator user ID |
| `created_at` | timestamp | Creation time |

**Message**

| Field | Type | Description |
|---|---|---|
| `id` | varchar(36) | UUID primary key |
| `channel_id` | varchar(36) | Indexed foreign key to channel |
| `user_id` | varchar(36) | Author user ID |
| `username` | varchar(64) | Author username (denormalized) |
| `content` | text | Message body |
| `created_at` | timestamp | Indexed creation time |

### Middleware Chain

| Middleware | Purpose |
|---|---|
| `gin.Recovery()` | Catches panics, returns 500 |
| `gzip.Gzip()` | Compresses responses (default level) |
| `OtelMiddleware()` | Creates OpenTelemetry spans per request |
| `LoggingMiddleware()` | Logs method, path, status, duration, IP, user-agent |
| `AuthMiddleware()` | Validates JWT Bearer token, sets `user_id`, `username`, `role` in context |
| `AdminMiddleware()` | Verifies admin role (falls back to DB query for legacy tokens) |
| `RateLimitMiddleware()` | Redis sliding-window rate limit per client IP |

---

## Makefile Commands

| Command | Description |
|---|---|
| `make build` | Build all three binaries (api, grpc, ws) to `bin/` |
| `make run` | Run the API server directly |
| `make start` | Run `start.bat` (Windows: Redis + API + Tunnel) |
| `make run-grpc` | Run standalone gRPC server |
| `make run-ws` | Run standalone WebSocket server |
| `make test` | Run all tests with verbose output |
| `make bench` | Run the built-in HTTP benchmark tool |
| `make hotreload` | Run with file-watcher auto-restart |
| `make proto` | Regenerate Protobuf Go code |
| `make clean` | Remove `bin/` and generated protobuf files |

---

## gRPC Service

The project includes a standalone gRPC server (`cmd/grpc/server.go`) with a sample Greeter service.

**Proto definition** (`proto/service.proto`):

```protobuf
service Greeter {
  rpc SayHello(HelloRequest) returns (HelloResponse);
}
```

Run with: `make run-grpc` (listens on port 9090 by default).

Features: keepalive (5 min max connection age), logging interceptor, panic recovery interceptor, health check service.

---

## License

This project is for educational and demonstration purposes.
