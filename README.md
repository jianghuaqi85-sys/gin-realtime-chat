# GIn -- 高性能 Gin 聊天服务器

## 项目功能
本项目可以在你自己电脑上部署一个聊天室，根据 Cloudflare Tunnel 所使用的服务器，你可以邀请你的朋友一起在这个聊天室里进行聊天。由于固定域名需要购买，我用的是其免费版，缺点是每次重启服务器都会更改域名，如果你有需求你可以购买一个固定的域名。这项目本身可以有很多的扩展方向比如导入文件图片等，你有兴趣的话可以根据其进一步开发使其完善。


基于 Go + Gin 框架的高性能实时聊天服务器，支持 WebSocket 消息推送、多频道聊天、管理面板、Redis Pub/Sub 水平扩展、Cloudflare Tunnel 一键公网穿透，内置单页聊天 UI。

---

## 技术栈

| 层级 | 技术 |
|---|---|
| 语言 | Go 1.25 |
| HTTP 框架 | Gin v1.10 |
| 数据库 | MySQL 8.x（GORM v1.31） |
| 缓存 / 消息总线 | Redis（go-redis v8） |
| WebSocket | Gorilla WebSocket v1.5 |
| 认证 | JWT（golang-jwt v5）+ bcrypt |
| 配置管理 | Viper（`.env` 文件 + 环境变量） |
| 日志 | Logrus（结构化 JSON） |
| 链路追踪 | OpenTelemetry（OTLP HTTP / stdout） |
| RPC | gRPC v1.80 + Protobuf |
| 压缩 | gin-contrib/gzip |
| 限流 | Redis 滑动窗口（Lua 脚本） |
| 公网穿透 | Cloudflare Tunnel（`cloudflared`） |
| 前端 | 单页 HTML/CSS/JS（内嵌在 Go 二进制中） |

---

## 功能特性

### 核心聊天

- 用户注册与登录（bcrypt 密码哈希）
- JWT 认证（HS256，可配置过期时间）
- 多频道聊天：创建、列出、切换频道
- WebSocket 实时消息推送，支持自动重连
- 消息编辑与删除（仅限自己的消息）
- 消息历史分页加载（基于时间戳游标）
- 用户加入/离开频道的系统消息
- 在线用户列表（定时刷新）
- 修改密码（需验证旧密码）
- 会话持久化（localStorage 自动登录）

### 管理面板

- 服务器统计面板（在线人数、注册用户、频道数、消息数）
- 用户管理：列表、删除、封禁、解封
- 频道管理：列表、删除、清空消息
- 全局广播公告
- 管理员路由权限保护（基于角色的中间件）

### 实时基础设施

- WebSocket Hub 256 分桶架构，降低锁竞争
- 64 分片频道锁，细粒度并发控制
- Write-behind 消息持久化：先广播再异步写库
- Redis Pub/Sub 消息总线，支持多实例水平扩展
- 优雅关闭：断开前通知所有客户端
- 指数退避重连 + HTTP 轮询降级（客户端侧）
- WebSocket 升级请求的 Origin 校验

### 性能优化

- Gzip 压缩中间件
- 数据库连接池调优：50 空闲 / 200 最大 / 1 小时生命周期
- Redis Lua 脚本原子化滑动窗口限流
- sync.Pool 复用日志字段 Map，减少 GC 压力
- 持久化 Worker 池（CPU 核数个 goroutine）异步写库
- FNV-32a 哈希分桶（客户端桶 + 频道锁）
- WebSocket 读写截止时间管理

### 可观测性

- OpenTelemetry 分布式链路追踪（OTLP HTTP 导出或 stdout）
- Logrus 结构化 JSON 日志（含请求元数据）
- 健康检查端点（`/api/public/health`）

### 部署

- Cloudflare Tunnel 集成（自动启动、URL 捕获、公网访问）
- ngrok 隧道 URL 自动检测（本地 API）
- Windows 一键启动脚本 `start.bat`（Redis + API + Tunnel）
- 内置热重载开发工具
- 内置 HTTP 压测工具

---

## 项目结构

```
GIn/
├── cmd/
│   ├── api/
│   │   ├── main.go            # API 服务器入口、路由注册、优雅关闭
│   │   └── chat_html.go       # 内嵌单页聊天 UI（HTML/CSS/JS）
│   ├── grpc/
│   │   └── server.go          # 独立 gRPC 服务器（Greeter 服务 + 健康检查）
│   └── ws/
│       └── server.go          # 独立 WebSocket 服务器
├── internal/
│   ├── config/
│   │   └── config.go          # Viper 配置加载器（含校验）
│   ├── database/
│   │   └── database.go        # GORM MySQL 连接（连接池调优）
│   ├── handler/
│   │   ├── auth_handler.go    # 登录、注册、获取当前用户、修改密码
│   │   ├── chat_handler.go    # 频道、消息、WebSocket 消息处理
│   │   └── admin_handler.go   # 管理统计、用户/频道/消息管理、广播
│   ├── logger/
│   │   └── logger.go          # Logrus 初始化与便捷封装
│   ├── middleware/
│   │   ├── auth.go            # JWT Bearer Token 认证中间件
│   │   ├── admin.go           # 管理员角色校验（兼容旧 Token 回退查库）
│   │   ├── rate_limit.go      # Redis 滑动窗口限流中间件
│   │   ├── logging.go         # 请求日志中间件（sync.Pool 优化）
│   │   └── otel.go            # OpenTelemetry Span 创建中间件
│   ├── repository/
│   │   ├── user_repository.go # 用户模型 + 内存/MySQL 双实现
│   │   └── chat_repository.go # 频道 + 消息模型 + MySQL 实现
│   ├── service/
│   │   └── greeter.go         # gRPC Greeter 服务实现
│   └── tunnel/
│       └── tunnel.go          # Cloudflare Tunnel 进程管理器
├── pkg/
│   ├── jwt/
│   │   └── jwt.go             # JWT 生成、解析、验证（HS256）
│   ├── limiter/
│   │   └── limiter.go         # Redis Lua 滑动窗口限流器
│   ├── otel/
│   │   └── otel.go            # OpenTelemetry Tracer Provider 初始化
│   ├── redisbus/
│   │   └── bus.go             # Redis Pub/Sub 跨实例消息总线
│   └── ws/
│       └── ws.go              # WebSocket Hub、Bucket、Client、频道分片
├── proto/
│   └── service.proto          # Greeter 服务 Protobuf 定义
├── tools/
│   ├── benchmark/
│   │   └── benchmark.go       # HTTP 压测工具（QPS、延迟、P99）
│   └── hotreload/
│       └── hotreload.go       # 文件监听自动重启工具
├── .env.example               # 环境配置模板
├── .gitignore                 # Git 忽略规则
├── go.mod                     # Go 模块定义
├── go.sum                     # 依赖校验和
├── Makefile                   # 构建、运行、测试命令
└── start.bat                  # Windows 一键启动脚本（Redis + API + Tunnel）
```

---

## 快速开始

### 环境要求

- Go 1.25+
- Redis 6.x+
- MySQL 8.x+（可选；不配置则使用内存存储，聊天功能不可用）
- `cloudflared` CLI（可选，用于公网穿透）

### 1. 克隆并配置

```bash
git clone https://github.com/your-username/GIn.git
cd GIn
cp .env.example .env
```

编辑 `.env` 文件（至少需要设置 `JWT_SECRET`，长度不少于 32 个字符）。

### 2. 安装依赖

```bash
go mod download
```

### 3. 启动

**方式一：Make 命令**

```bash
make run          # 直接运行 API 服务器
make build        # 编译所有二进制到 bin/ 目录
make start        # 运行 start.bat（Windows）
```

**方式二：Go 直接运行**

```bash
go run ./cmd/api/
```

**方式三：Windows 一键启动**

双击 `start.bat`，自动完成：
1. 启动 Redis（如果未运行）
2. 启动 API 服务器
3. 启动 Cloudflare Tunnel
4. 显示本地和公网访问地址

### 4. 访问

- 本地：`http://localhost:8080`
- 公网：查看 Cloudflare Tunnel 窗口或 UI 侧边栏中的隧道地址

### 5. 默认管理员账号

| 字段 | 值 |
|---|---|
| 用户名 | `admin` |
| 密码 | `admin123` |

首次登录后请立即修改密码。

---

## API 文档

基础 URL：`http://localhost:8080`

所有需要认证的接口必须在请求头中携带 `Authorization: Bearer <token>`。

### 公开接口（无需认证）

| 方法 | 路径 | 说明 | 限流 |
|---|---|---|---|
| `POST` | `/api/public/register` | 用户注册 | 20次/分钟/IP |
| `POST` | `/api/public/login` | 用户登录，返回 JWT Token | 20次/分钟/IP |
| `GET` | `/api/public/health` | 健康检查 | 20次/分钟/IP |

#### POST /api/public/register

```json
// 请求
{ "username": "alice", "password": "secret123", "confirm_password": "secret123" }

// 成功 201
{ "message": "user created" }

// 错误 409
{ "error": "用户名已被占用" }
```

校验规则：
- 用户名：3-32 个字符，仅允许字母、数字、下划线
- 密码：至少 8 个字符
- 确认密码必须一致

#### POST /api/public/login

```json
// 请求
{ "username": "alice", "password": "secret123" }

// 成功 200
{ "token": "eyJhbGciOiJIUzI1NiIs..." }

// 错误 401
{ "error": "用户名或密码错误" }
```

#### GET /api/public/health

```json
// 成功 200
{ "status": "ok", "timestamp": 1715000000 }
```

### 认证接口

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/api/me` | 获取当前用户信息 |
| `PUT` | `/api/password` | 修改密码 |
| `GET` | `/api/channels` | 获取频道列表 |
| `POST` | `/api/channels` | 创建频道 |
| `GET` | `/api/channels/:id/messages` | 获取频道消息（分页） |
| `PUT` | `/api/messages/:id` | 编辑自己的消息 |
| `DELETE` | `/api/messages/:id` | 删除自己的消息 |
| `GET` | `/api/online` | 获取在线用户列表 |
| `GET` | `/api/tunnel` | 获取公网隧道地址 |

认证接口限流：100 次/分钟/IP。

#### GET /api/me

```json
// 成功 200
{ "user_id": "uuid...", "username": "alice" }
```

#### PUT /api/password

```json
// 请求
{ "old_password": "secret123", "new_password": "newsecret456" }

// 成功 200
{ "message": "密码修改成功" }
```

#### POST /api/channels

```json
// 请求
{ "name": "general" }

// 成功 201
{ "ID": "uuid...", "Name": "general", "CreatedBy": "uuid...", "CreatedAt": "..." }
```

#### GET /api/channels/:id/messages

查询参数：
- `limit`（默认：50，最大：200）— 返回消息数量
- `before`（可选，RFC3339 时间戳）— 分页游标

```json
// 成功 200
[
  {
    "ID": "uuid...",
    "ChannelID": "uuid...",
    "UserID": "uuid...",
    "Username": "alice",
    "Content": "你好！",
    "CreatedAt": "2026-05-12T10:00:00Z"
  }
]
```

#### PUT /api/messages/:id

```json
// 请求
{ "content": "修改后的消息内容" }

// 成功 200
{ "message": "编辑成功" }
```

#### DELETE /api/messages/:id

```json
// 成功 200
{ "message": "删除成功" }

// 错误 403
{ "error": "只能删除自己的消息" }
```

#### GET /api/tunnel

```json
// 成功 200
{ "urls": ["https://random-name.trycloudflare.com"] }

// 错误 404
{ "error": "没有检测到隧道，请确认 ngrok 或 Cloudflare Tunnel 已启动" }
```

### 管理员接口

所有管理员接口需要认证 + 管理员角色。

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/api/admin/stats` | 服务器统计 |
| `GET` | `/api/admin/users` | 用户列表 |
| `DELETE` | `/api/admin/users/:id` | 删除用户（先断开连接） |
| `POST` | `/api/admin/ban` | 封禁用户 |
| `POST` | `/api/admin/unban` | 解封用户 |
| `DELETE` | `/api/admin/channels/:id` | 删除频道及其消息 |
| `DELETE` | `/api/admin/channels/:id/messages` | 清空频道消息 |
| `DELETE` | `/api/admin/messages/:id` | 删除任意消息 |
| `POST` | `/api/admin/broadcast` | 向所有用户发送系统公告 |

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
// 请求
{ "user_id": "uuid..." }

// 成功 200
{ "banned": "uuid..." }
```

#### POST /api/admin/unban

```json
// 请求
{ "user_id": "uuid..." }

// 成功 200
{ "unbanned": "uuid..." }
```

#### POST /api/admin/broadcast

```json
// 请求
{ "content": "系统维护通知：今晚 22:00 将进行升级" }

// 成功 200
{ "broadcast": "系统维护通知：今晚 22:00 将进行升级" }
```

---

## WebSocket 协议

端点：`ws://localhost:8080/api/ws`（或通过 TLS 使用 `wss://`）

### 连接流程

```
客户端                              服务器
  |                                   |
  |--- WebSocket 升级请求 ------------>|
  |<-- 101 切换协议 -------------------|
  |                                   |
  |--- { type: "auth", token } ------>|   （必须是第一条消息，10 秒超时）
  |<-- { type: "auth_ok" } -----------|
  |                                   |
  |--- { type: "join", channel_id } ->|
  |<-- { type: "system", content: "alice 加入了频道" } --|
  |                                   |
  |--- { type: "message", channel_id, content } ->|
  |<-- { type: "message", ... } ------|  （广播给频道内所有成员）
  |                                   |
  |--- { type: "leave", channel_id }->|
  |<-- { type: "system", content: "alice 离开了频道" } --|
```

### 消息类型

#### 客户端 -> 服务器

| type | 字段 | 说明 |
|---|---|---|
| `auth` | `token` | JWT 认证，必须是第一条消息 |
| `join` | `channel_id` | 加入频道以接收消息 |
| `leave` | `channel_id` | 离开频道 |
| `message` | `channel_id`, `content` | 向已加入的频道发送消息 |

#### 服务器 -> 客户端

| type | 字段 | 说明 |
|---|---|---|
| `auth_ok` | `content`（用户名） | 认证成功 |
| `error` | `content` | 错误消息（认证失败、未加入频道等） |
| `message` | `channel_id`, `user_id`, `username`, `content`, `created_at` | 频道新消息 |
| `system` | `channel_id`（可选）, `content`, `created_at` | 系统通知（加入/离开/广播） |

### 连接参数

| 参数 | 值 |
|---|---|
| Ping 间隔 | 54 秒 |
| Pong 超时 | 60 秒 |
| 写入截止时间 | 10 秒 |
| 认证超时 | 10 秒 |
| 默认读取限制 | 512 字节（可通过 `WS_READ_LIMIT` 配置） |
| 每客户端发送缓冲 | 256 条消息 |

### 客户端重连策略

内置 UI 实现了：
1. 指数退避：1s → 2s → 4s → 8s → 16s → 30s（上限）
2. 连续 5 次失败后降级为 HTTP 轮询（每 5 秒一次）
3. WebSocket 恢复后自动切回

---

## 配置说明

所有配置从 `.env` 文件和/或环境变量加载，环境变量优先。

| 变量 | 类型 | 默认值 | 说明 |
|---|---|---|---|
| `APP_ENV` | string | `development` | 应用环境（`development` / `production`） |
| `APP_PORT` | string | `8080` | HTTP API 服务器端口 |
| `GRPC_PORT` | string | `9090` | gRPC 服务器端口（独立模式） |
| `WS_PORT` | string | `8081` | WebSocket 服务器端口（独立模式） |
| `JWT_SECRET` | string | **（必填）** | JWT 签名密钥，至少 32 个字符 |
| `JWT_EXPIRE_HOURS` | int | `24` | JWT Token 过期时间（小时） |
| `REDIS_ADDR` | string | `localhost:6379` | Redis 服务器地址 |
| `REDIS_PASSWORD` | string | *(空)* | Redis 认证密码 |
| `REDIS_DB` | int | `0` | Redis 数据库编号 |
| `MYSQL_DSN` | string | *(空)* | MySQL 连接字符串。留空则使用内存存储，聊天功能不可用 |
| `OTEL_ENDPOINT` | string | *(空)* | OpenTelemetry 采集器端点。设为 `stdout` 或留空输出到控制台 |
| `OTEL_INSECURE` | bool | `true` | 允许不安全的 OTLP HTTP 连接 |
| `LOG_LEVEL` | string | `info` | 日志级别：`trace`, `debug`, `info`, `warn`, `error`, `fatal` |
| `DB_LOG_LEVEL` | string | `warn` | GORM SQL 日志级别：`silent`, `error`, `warn`, `info` |
| `WS_ALLOWED_ORIGIN` | string | *(空)* | 允许的 WebSocket Origin。留空则只允许同源连接 |
| `WS_READ_LIMIT` | int | `512` | WebSocket 消息最大字节数 |
| `CLOUDFLARED_PATH` | string | *(空)* | `cloudflared` 可执行文件路径。设置后启动时自动开启 Cloudflare Tunnel |

### MySQL DSN 格式

```
user:password@tcp(host:port)/dbname?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local
```

---

## 部署说明

### 本地开发

```bash
# 终端 1：启动 Redis
redis-server

# 终端 2：启动 API 服务器
make run

# 或使用热重载开发
make hotreload
```

### 生产环境构建

```bash
make build
# 输出：
#   bin/api   - HTTP API + WebSocket + 内嵌 UI
#   bin/grpc  - 独立 gRPC 服务器
#   bin/ws    - 独立 WebSocket 服务器
```

### Docker Compose（MySQL + Redis）

创建 `docker-compose.yml` 管理依赖：

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

**自动模式（推荐）：**

在 `.env` 中设置 `CLOUDFLARED_PATH` 指向 `cloudflared` 可执行文件路径。服务器启动时会自动开启隧道并捕获公网地址。

```env
CLOUDFLARED_PATH=/usr/local/bin/cloudflared
```

**手动模式：**

```bash
cloudflared tunnel --url http://localhost:8080
```

隧道地址会显示在服务器日志和聊天 UI 侧边栏中，同时保存到 `.tunnel_url` 文件供 API 读取。

**ngrok（替代方案）：**

如果 ngrok 在默认端口（4040）运行，服务器会自动通过 ngrok 本地 API 检测隧道地址。

### Windows 一键启动

编辑 `start.bat` 配置路径：

```bat
set CLOUDFLARED=D:\cloudflared.exe
set REDIS_SERVER=D:\Redis\redis-server.exe
set REDIS_CONFIG=D:\Redis\redis.windows.conf
```

双击 `start.bat`，自动完成：
1. 启动 Redis（如果未运行）
2. 启动 API 服务器
3. 启动 Cloudflare Tunnel
4. 显示本地和公网访问地址

---

## 架构说明

### 请求流程

```
客户端请求
    |
    v
[Gin 路由器]
    |
    +-- Recovery 中间件（panic 恢复）
    +-- Gzip 中间件（响应压缩）
    +-- OTel 中间件（分布式链路追踪）
    +-- Logging 中间件（结构化请求日志）
    |
    +-- /api/public/*  --> 限流中间件（20次/分钟） --> 认证处理器
    +-- /api/*         --> 限流中间件（100次/分钟） --> 认证中间件 --> 业务处理器
    +-- /api/admin/*   --> 限流中间件（100次/分钟） --> 认证中间件 --> 管理员中间件 --> 管理处理器
    +-- /api/ws        --> WebSocket 升级 --> 认证（第一条消息） --> Hub
```

### WebSocket 架构

```
                    +------------------+
                    |     Hub          |
                    |  （256 个桶）     |
                    +--------+---------+
                             |
            +----------------+----------------+
            |                |                |
     +------+------+  +-----+------+  +------+------+
     |   桶 0      |  |   桶 1      |  |  桶 255     |
     | (goroutine) |  | (goroutine) |  | (goroutine) |
     +------+------+  +-----+------+  +------+------+
            |                |                |
       +----+----+      +----+----+      +----+----+
       | 客户端  |      | 客户端  |      | 客户端  |
       | 客户端  |      | 客户端  |      | 客户端  |
       +---------+      +---------+      +---------+

频道分片（64 个）：
  shard[0]: { 频道A: [客户端1, 客户端2], 频道B: [客户端3] }
  shard[1]: { 频道C: [客户端4] }
  ...
  shard[63]: { 频道Z: [客户端N] }
```

关键设计决策：
- **256 哈希分桶客户端池**：按用户 ID 的 FNV-32a 哈希分配桶，每桶独立 goroutine，降低锁竞争
- **64 分片频道锁**：按频道 ID 的 FNV-32a 哈希分配分片，支持频道操作并发
- **Write-behind 持久化**：消息先广播再入队（容量 4096 的 channel），由 Worker 池（CPU 核数个，最少 2 个）异步写库

### 多实例扩展（Redis Pub/Sub）

```
  实例 A                        实例 B
  +----------+                  +----------+
  | Hub      |                  | Hub      |
  | 客户端 1 |                  | 客户端 3 |
  | 客户端 2 |                  | 客户端 4 |
  +----+-----+                  +----+-----+
       |                             |
       v                             v
  +----+-----------------------------+-----+
  |           Redis Pub/Sub                |
  |  频道：chat:ch:<channel_id>            |
  +----------------------------------------+
```

配置 `REDIS_ADDR` 后，消息总线自动启用 Redis Pub/Sub。发布到频道的消息会广播到所有订阅该频道的实例。Hub 根据本地客户端计数管理订阅/取消订阅的生命周期。

### 数据模型

**用户（User）**

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | varchar(36) | UUID 主键 |
| `username` | varchar(64) | 唯一，字母数字下划线 |
| `password_hash` | varchar(255) | bcrypt 哈希 |
| `role` | varchar(16) | `admin` 或 `user` |
| `banned` | boolean | 封禁标志 |

**频道（Channel）**

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | varchar(36) | UUID 主键 |
| `name` | varchar(64) | 唯一频道名 |
| `created_by` | varchar(36) | 创建者用户 ID |
| `created_at` | timestamp | 创建时间 |

**消息（Message）**

| 字段 | 类型 | 说明 |
|---|---|---|
| `id` | varchar(36) | UUID 主键 |
| `channel_id` | varchar(36) | 索引外键，关联频道 |
| `user_id` | varchar(36) | 作者用户 ID |
| `username` | varchar(64) | 作者用户名（冗余字段） |
| `content` | text | 消息内容 |
| `created_at` | timestamp | 索引创建时间 |

### 中间件链

| 中间件 | 作用 |
|---|---|
| `gin.Recovery()` | 捕获 panic，返回 500 |
| `gzip.Gzip()` | 响应压缩（默认级别） |
| `OtelMiddleware()` | 为每个请求创建 OpenTelemetry Span |
| `LoggingMiddleware()` | 记录请求方法、路径、状态码、耗时、IP、User-Agent |
| `AuthMiddleware()` | 验证 JWT Bearer Token，将 `user_id`、`username`、`role` 存入 Context |
| `AdminMiddleware()` | 校验管理员角色（旧 Token 无 role 时回退查库） |
| `RateLimitMiddleware()` | Redis 滑动窗口限流（按客户端 IP） |

---

## Makefile 命令

| 命令 | 说明 |
|---|---|
| `make build` | 编译三个二进制文件（api、grpc、ws）到 `bin/` |
| `make run` | 直接运行 API 服务器 |
| `make start` | 运行 `start.bat`（Windows：Redis + API + Tunnel） |
| `make run-grpc` | 运行独立 gRPC 服务器 |
| `make run-ws` | 运行独立 WebSocket 服务器 |
| `make test` | 运行所有测试（详细输出） |
| `make bench` | 运行内置 HTTP 压测工具 |
| `make hotreload` | 使用文件监听热重载运行 |
| `make proto` | 重新生成 Protobuf Go 代码 |
| `make clean` | 删除 `bin/` 目录和生成的 Protobuf 文件 |

---

## gRPC 服务

项目包含一个独立 gRPC 服务器（`cmd/grpc/server.go`），内置示例 Greeter 服务。

**Proto 定义**（`proto/service.proto`）：

```protobuf
service Greeter {
  rpc SayHello(HelloRequest) returns (HelloResponse);
}
```

运行：`make run-grpc`（默认监听 9090 端口）。

特性：Keepalive（最大连接时长 5 分钟）、日志拦截器、Panic 恢复拦截器、健康检查服务。

---

## 开源协议

无
