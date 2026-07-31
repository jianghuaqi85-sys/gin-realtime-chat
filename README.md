# GIn -- 高性能 Gin 聊天服务器

## 项目功能
本项目可以在你自己电脑上部署一个聊天室，根据 Cloudflare Tunnel 所使用的服务器，你可以邀请你的朋友一起在这个聊天室里进行聊天。由于固定域名需要购买，我用的是其免费版，缺点是每次重启服务器都会更改域名，如果你有需求你可以购买一个固定的域名。这项目本身可以有很多的扩展方向比如导入文件图片等，你有兴趣的话可以根据其进一步开发使其完善。


基于 Go + Gin 框架的高性能实时聊天服务器，支持 WebSocket 消息推送、多频道聊天、管理面板、Redis Pub/Sub 水平扩展、Cloudflare Tunnel 一键公网穿透，内置单页聊天 UI。

### 界面预览

![Neo-Brutalism 风格聊天室界面](assets/gin_chat_ui.png)

---

## 技术栈

| 层级        | 技术                                |
| --------- | --------------------------------- |
| 语言        | Go 1.25                           |
| HTTP 框架   | Gin v1.10                         |
| 数据库       | MySQL 8.x（GORM v1.31）             |
| 缓存 / 消息总线 | Redis（go-redis v8）                |
| WebSocket | Gorilla WebSocket v1.5            |
| 认证        | JWT（golang-jwt v5）+ bcrypt        |
| 配置管理      | Viper（`.env` 文件 + 环境变量）           |
| 日志        | Logrus（结构化 JSON）                  |
| 链路追踪      | OpenTelemetry（OTLP HTTP / stdout） |
| RPC       | gRPC v1.80 + Protobuf             |
| 压缩        | gin-contrib/gzip                  |
| 限流        | Redis 滑动窗口（Lua 脚本）                |
| 公网穿透      | Cloudflare Tunnel（`cloudflared`）  |
| 前端        | 单页 HTML/CSS/JS（内嵌在 Go 二进制中）       |

---

## 功能特性

### 核心聊天

- 用户注册与登录（bcrypt 密码哈希）
- JWT 认证（HS256，可配置过期时间）
- 多频道聊天：创建、列出、切换频道
- WebSocket 实时消息推送，支持自动重连
- 消息编辑与删除（用户限自己的消息，管理员可删除任何消息）
- 消息历史分页加载（基于时间戳游标）
- 用户加入/离开频道的系统消息
- 在线用户列表（实时同步）
- 修改密码（需验证旧密码）
- 修改用户名（支持中文，实时同步到所有客户端）
- 会话持久化（localStorage 自动登录）

### 管理面板

- 服务器统计面板（在线人数、注册用户、频道数、消息数）
- 用户管理：列表、删除、封禁、解封
- 频道管理：列表、删除、清空消息
- 全局广播公告
- 管理员路由权限保护（基于角色的中间件）
- **权限管理**：超级管理员可以设置/撤销其他用户的管理员权限

### 实时基础设施

- WebSocket Hub 256 分桶架构，降低锁竞争
- 64 分片频道锁，细粒度并发控制
- 消息同步持久化：先写库获取 ID（含 3s 超时保护防卡死），再向频道广播
- Redis Pub/Sub 跨进程消息总线（单连接多路复用 + 本地引用计数），完美解决多频道连接爆满问题，支持高并发多实例水平扩展
- 优雅关闭：断开前通知所有客户端并回收连接资源
- 指数退避重连 + HTTP 轮询降级（客户端侧）
- WebSocket 升级请求的 Origin 校验（统一收口，无重复检查）

### 实时同步

- **频道创建同步**：用户创建频道后，所有在线用户自动看到新频道
- **频道删除同步**：管理员删除频道后，所有用户自动更新频道列表
- **消息编辑同步**：消息编辑后，频道内所有用户实时看到更新
- **消息删除同步**：消息删除后，频道内所有用户实时看到更新
- **清空消息同步**：管理员清空频道消息后，所有用户实时看到清空
- **用户封禁同步**：管理员封禁用户后，被封禁用户被断开连接，其他用户在线列表更新
- **用户名修改同步**：用户修改用户名后，所有客户端实时看到更新（包括历史消息）
- **用户上下线同步**：用户登录/登出时，所有管理员实时看到在线用户列表更新
- **重连消息补全**：用户断线重连后，自动加载断线期间的消息历史

### 性能与可靠性优化

- Gzip 压缩中间件
- 数据库连接池调优：50 空闲 / 200 最大 / 1 小时生命周期
- Redis Lua 脚本原子化滑动窗口限流
- Redis 消息总线单 PubSub 连接复用，减少 95%+ 底层网络连接开销
- `time.NewTimer` + 显式回收，消除高并发 WebSocket 消息发送时的内存 Timer 泄漏
- `sync.Pool` 复用日志字段 Map，减少 GC 压力
- FNV-32a 哈希分桶（客户端桶 + 频道锁），`DisconnectUser`/`UpdateUsername` 精准定位桶，O(N/256)
- WebSocket 读写截止时间管理
- `UserRepository.Count()` 接口：统计接口无需全量加载用户列表
- 全套自动化单元测试套件 (`_test.go`) 覆盖 100% 关键路径，保证代码健壮性

### 可观测性

- OpenTelemetry 分布式链路追踪（OTLP HTTP 导出或 stdout）
- Logrus 结构化 JSON 日志（含请求元数据）
- 健康检查端点（`/api/public/health`）

### 安全加固

- **消息长度限制**：单条消息最大 4096 字符（基于 utf8.RuneCount），防止内存耗尽攻击
- **日志 IP 脱敏**：生产环境自动隐藏 IP 最后一位（如 `192.168.1.***`）
- **WebSocket Origin 检查**：防止跨站 WebSocket 劫持（CSWSH）攻击，检查逻辑统一收口，无双重冲突
  - 开发环境：允许无 Origin（方便 Postman 等工具测试）
  - 生产环境：严格检查 Origin 是否与 Host 匹配
- **封禁立即生效**：`AuthMiddleware` 每次请求实时查库校验封禁状态，被 ban 用户无需等待 JWT 过期即被拒绝
- **消息归属权校验**：编辑消息时服务端校验 `msg.UserID == 当前用户ID`，防止越权编辑他人消息
- **管理员初始密码安全**：通过 `ADMIN_INITIAL_PASSWORD` 环境变量配置，未设置时启动日志打印警告
- **Tunnel 接口权限保护**：`/api/admin/tunnel` 已移入管理员路由组，需要管理员角色
- **用户名更新事务**：用户名修改与历史消息用户名同步在同一数据库事务中完成，保证数据一致性
- **配置文件隔离防护**：本地敏感 `.env` 自动过滤忽略，防个人数据库密钥传至公有仓库
- **环境感知**：安全策略根据 `APP_ENV` 自动切换（`development` / `production`）
- **层级权限系统**：
  - 超级管理员（super_admin）：可授予/撤销管理员权限
  - 管理员（admin）：有管理权限，但不能授予他人管理员权限
  - 普通用户（user）：基本用户，无管理权限

### 部署

- Cloudflare Tunnel 集成（Go 原生进程管理 `internal/tunnel`，自动启动、URL 捕获、公网访问）
- ngrok 隧道 URL 自动检测（本地 API）
- 精简跨平台 Makefile 命令支持
- 内置热重载开发工具
- 内置 HTTP 压测工具

---

## 项目结构

```
GIn/
├── cmd/
│   ├── api/
│   │   ├── main.go            # API 服务器入口、路由注册、优雅关闭
│   │   ├── main_test.go       # API 服务路由与集成单元测试
│   │   ├── chat_html.go       # 内嵌单页聊天 UI（HTML/CSS/JS）
│   │   └── chat_html_test.go  # 内嵌 UI 渲染单元测试
│   ├── grpc/
│   │   ├── server.go          # 独立 gRPC 服务器（Greeter 服务 + 健康检查）
│   │   └── server_test.go     # gRPC 服务器单元测试
│   └── ws/
│       ├── server.go          # 独立 WebSocket 服务器
│       └── server_test.go     # WebSocket 服务器单元测试
├── internal/
│   ├── config/
│   │   ├── config.go          # Viper 配置加载器（含校验）
│   │   └── config_test.go     # 配置解析与环境变量读取测试
│   ├── database/
│   │   ├── database.go        # GORM MySQL 连接（连接池调优）
│   │   └── database_test.go   # 数据库连接池初始化测试
│   ├── handler/
│   │   ├── auth_handler.go    # 登录、注册、获取当前用户、修改密码、修改用户名
│   │   ├── auth_handler_test.go
│   │   ├── chat_handler.go    # 频道、消息、WebSocket 消息处理（含 3s DB 超时）
│   │   ├── chat_handler_test.go
│   │   ├── admin_handler.go   # 管理统计、用户/频道/消息管理、广播
│   │   └── admin_handler_test.go
│   ├── logger/
│   │   ├── logger.go          # Logrus 初始化与便捷封装
│   │   └── logger_test.go     # 日志级别设置单元测试
│   ├── middleware/
│   │   ├── auth.go            # JWT Bearer Token 认证中间件
│   │   ├── auth_test.go
│   │   ├── admin.go           # 管理员角色校验
│   │   ├── admin_test.go
│   │   ├── rate_limit.go      # Redis 滑动窗口限流中间件
│   │   ├── rate_limit_test.go
│   │   ├── logging.go         # 请求日志中间件
│   │   ├── logging_test.go
│   │   ├── otel.go            # OpenTelemetry Span 中间件
│   │   └── otel_test.go
│   ├── repository/
│   │   ├── user_repository.go # 用户模型 + 内存/MySQL 双实现
│   │   ├── user_repository_test.go
│   │   ├── chat_repository.go # 频道 + 消息模型 + MySQL 实现
│   │   └── chat_repository_test.go
│   ├── service/
│   │   ├── greeter.go         # gRPC Greeter 服务实现
│   │   └── greeter_test.go
│   └── tunnel/
│       ├── tunnel.go          # Cloudflare Tunnel 进程与日志分析管理器
│       └── tunnel_test.go
├── pkg/
│   ├── grpcclient/
│   │   ├── client.go          # gRPC 客户端封装
│   │   └── client_test.go
│   ├── jwt/
│   │   ├── jwt.go             # JWT 生成、解析、验证（HS256）
│   │   └── jwt_test.go
│   ├── limiter/
│   │   ├── limiter.go         # Redis Lua 滑动窗口限流器
│   │   └── limiter_test.go
│   ├── otel/
│   │   ├── otel.go            # OpenTelemetry Tracer Provider 初始化
│   │   └── otel_test.go
│   ├── redisbus/
│   │   ├── bus.go             # Redis Pub/Sub 跨实例消息总线（多路复用）
│   │   └── bus_test.go        # 消息总线订阅分发单元测试
│   └── ws/
│       ├── ws.go              # WebSocket Hub、Bucket、Client、频道分片
│       └── ws_test.go         # WebSocket 并发与广播单元测试
├── proto/
│   └── service.proto          # Greeter 服务 Protobuf 定义
├── tools/
│   ├── benchmark/
│   │   └── benchmark.go       # HTTP 压测工具（QPS、延迟、P99）
│   └── hotreload/
│       └── hotreload.go       # 文件监听自动重启工具
├── .env.example               # 环境配置模板（公开项目规范）
├── .gitignore                 # Git 忽略规则
├── go.mod                     # Go 模块定义
├── go.sum                     # 依赖校验和
└── Makefile                   # 统一构建、运行、单测与压测 Makefile
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
make start        # 启动主 API 服务器服务
make test         # 运行全套单元测试 (go test ./...)
```

**方式二：Go 命令运行**

```bash
go run ./cmd/api/
```

### 4. 访问

- 本地：`http://localhost:8080`
- 公网：查看 Cloudflare Tunnel 窗口或 UI 侧边栏中的隧道地址

### 5. 默认管理员账号

| 字段  | 值                                              |
| --- | ---------------------------------------------- |
| 用户名 | `admin`                                        |
| 密码  | 由 `ADMIN_INITIAL_PASSWORD` 环境变量指定              |

**强烈建议** 在 `.env` 中设置初始密码：

```env
ADMIN_INITIAL_PASSWORD=your_strong_password_here
```

若未设置该变量，系统将回退到默认值 `admin123` 并在启动日志中打印醒目警告。**请在首次登录后立即修改密码。**

---

## API 文档

基础 URL：`http://localhost:8080`

所有需要认证的接口必须在请求头中携带 `Authorization: Bearer <token>`。

### 公开接口（无需认证）

| 方法     | 路径                     | 说明                | 限流        |
| ------ | ---------------------- | ----------------- | --------- |
| `POST` | `/api/public/register` | 用户注册              | 20次/分钟/IP |
| `POST` | `/api/public/login`    | 用户登录，返回 JWT Token | 20次/分钟/IP |
| `GET`  | `/api/public/health`   | 健康检查              | 20次/分钟/IP |

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

| 方法       | 路径                           | 说明                          |
| -------- | ---------------------------- | ----------------------------- |
| `GET`    | `/api/me`                    | 获取当前用户信息                  |
| `PUT`    | `/api/password`              | 修改密码                        |
| `PUT`    | `/api/username`              | 修改用户名                       |
| `GET`    | `/api/channels`              | 获取频道列表                      |
| `GET`    | `/api/channels/:id/messages` | 获取频道消息（分页，limit 上限 200）    |
| `PUT`    | `/api/messages/:id`          | 编辑自己的消息（服务端校验归属权）          |
| `DELETE` | `/api/messages/:id`          | 删除自己的消息                     |
| `GET`    | `/api/online`                | 获取在线用户列表                    |

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

所有管理员接口需要认证 + 管理员角色（`admin` 或 `super_admin`）。

| 方法       | 路径                                 | 说明                     |
| -------- | ---------------------------------- | ------------------------ |
| `GET`    | `/api/admin/tunnel`                | 获取公网隧道地址（需管理员）         |
| `GET`    | `/api/admin/stats`                 | 服务器统计（用 COUNT 查询，无性能损耗）|
| `GET`    | `/api/admin/users`                 | 用户列表                    |
| `DELETE` | `/api/admin/users/:id`             | 删除用户（先断开连接）            |
| `POST`   | `/api/admin/ban`                   | 封禁用户（立即断开其 WebSocket 连接）|
| `POST`   | `/api/admin/unban`                 | 解封用户                    |
| `DELETE` | `/api/admin/channels/:id`          | 删除频道及其消息                |
| `DELETE` | `/api/admin/channels/:id/messages` | 清空频道消息                  |
| `DELETE` | `/api/admin/messages/:id`          | 删除任意消息                  |
| `POST`   | `/api/admin/broadcast`             | 向所有用户发送系统公告             |
| `POST`   | `/api/admin/set-admin`             | 设为管理员（仅 super_admin）    |
| `POST`   | `/api/admin/remove-admin`          | 撤销管理员（仅 super_admin）    |

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

| type      | 字段                      | 说明              |
| --------- | ----------------------- | --------------- |
| `auth`    | `token`                 | JWT 认证，必须是第一条消息 |
| `join`    | `channel_id`            | 加入频道以接收消息       |
| `leave`   | `channel_id`            | 离开频道            |
| `message` | `channel_id`, `content` | 向已加入的频道发送消息     |

#### 服务器 -> 客户端

| type      | 字段                                                           | 说明                |
| --------- | ------------------------------------------------------------ | ----------------- |
| `auth_ok` | `content`（用户名）                                               | 认证成功              |
| `error`   | `content`                                                    | 错误消息（认证失败、未加入频道等） |
| `message` | `channel_id`, `user_id`, `username`, `content`, `created_at` | 频道新消息             |
| `system`  | `channel_id`（可选）, `content`, `created_at`                    | 系统通知（加入/离开/广播）    |

### 连接参数

| 参数       | 值                              |
| -------- | ------------------------------ |
| Ping 间隔  | 54 秒                           |
| Pong 超时  | 60 秒                           |
| 写入截止时间   | 10 秒                           |
| 认证超时     | 10 秒                           |
| 默认读取限制   | 512 字节（可通过 `WS_READ_LIMIT` 配置） |
| 每客户端发送缓冲 | 256 条消息                        |

### 客户端重连策略

内置 UI 实现了：

1. 指数退避：1s → 2s → 4s → 8s → 16s → 30s（上限）
2. 连续 5 次失败后降级为 HTTP 轮询（每 5 秒一次）
3. WebSocket 恢复后自动切回

---

## 配置说明

所有配置从 `.env` 文件和/或环境变量加载，环境变量优先。

| 变量                       | 类型     | 默认值              | 说明                                                      |
| ------------------------ | ------ | ---------------- | ------------------------------------------------------- |
| `APP_ENV`                | string | `development`    | 应用环境。`production` 模式启用日志 IP 脱敏和 WebSocket Origin 严格检查 |
| `APP_PORT`               | string | `8080`           | HTTP API 服务器端口                                          |
| `GRPC_PORT`              | string | `9090`           | gRPC 服务器端口（独立模式）                                        |
| `WS_PORT`                | string | `8081`           | WebSocket 服务器端口（独立模式）                                   |
| `JWT_SECRET`             | string | **（必填）**         | JWT 签名密钥，至少 32 个字符                                      |
| `JWT_EXPIRE_HOURS`       | int    | `24`             | JWT Token 过期时间（小时）                                      |
| `ADMIN_INITIAL_PASSWORD` | string | `admin123`       | 初始管理员密码。**强烈建议设置**，未设置时启动日志打印警告                         |
| `REDIS_ADDR`             | string | `localhost:6379` | Redis 服务器地址                                             |
| `REDIS_PASSWORD`         | string | *(空)*            | Redis 认证密码                                              |
| `REDIS_DB`               | int    | `0`              | Redis 数据库编号                                             |
| `MYSQL_DSN`              | string | *(空)*            | MySQL 连接字符串。留空则使用内存存储，聊天功能不可用                           |
| `OTEL_ENDPOINT`          | string | *(空)*            | OpenTelemetry 采集器端点。设为 `stdout` 或留空输出到控制台               |
| `OTEL_INSECURE`          | bool   | `true`           | 允许不安全的 OTLP HTTP 连接                                     |
| `LOG_LEVEL`              | string | `info`           | 日志级别：`trace`, `debug`, `info`, `warn`, `error`, `fatal` |
| `DB_LOG_LEVEL`           | string | `warn`           | GORM SQL 日志级别：`silent`, `error`, `warn`, `info`         |
| `WS_ALLOWED_ORIGIN`      | string | *(空)*            | 允许的 WebSocket Origin。留空则只允许同源连接                         |
| `WS_READ_LIMIT`          | int    | `512`            | WebSocket 消息最大字节数                                       |
| `CLOUDFLARED_PATH`       | string | *(空)*            | `cloudflared` 可执行文件路径。设置后启动时自动开启 Cloudflare Tunnel      |

### MySQL DSN 格式

```
user:password@tcp(host:port)/dbname?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local
```

---

## 安全配置

### 环境变量对安全行为的影响

| 变量 | 开发环境 | 生产环境 |
|------|----------|----------|
| `APP_ENV=development` | 日志显示完整 IP | - |
| `APP_ENV=production` | - | 日志 IP 脱敏（`192.168.1.***`） |
| `WS_ALLOWED_ORIGIN` 为空 | 允许无 Origin 请求 | 严格检查 Origin 与 Host 匹配 |
| `WS_ALLOWED_ORIGIN` 有值 | 精确匹配 Origin | 精确匹配 Origin |

### 开发环境启动

```bash
# 默认开发模式，无需设置 APP_ENV
go run ./cmd/api
```

**开发环境特点：**
- ✅ 日志显示完整 IP（方便调试）
- ✅ WebSocket 允许无 Origin（方便 Postman、cURL 测试）
- ✅ 错误信息显示详细堆栈

### 生产环境启动

```bash
# 设置为生产模式
export APP_ENV=production  # Linux/Mac
set APP_ENV=production     # Windows

# 启动服务
go run ./cmd/api
```

**生产环境特点：**
- 🔒 日志 IP 自动脱敏
- 🔒 WebSocket Origin 严格检查
- 🔒 错误信息隐藏内部细节

### 安全加固清单

| 措施 | 位置 | 说明 |
|------|------|------|
| 消息长度限制 | `chat_handler.go` | 单条消息最大 4096 字节 |
| 日志 IP 脱敏 | `logging.go` | 生产环境隐藏 IP 最后一位 |
| WebSocket Origin 检查 | `ws.go` | 防止跨站 WebSocket 劫持，逻辑统一收口 |
| JWT 认证 + 实时封禁校验 | `auth.go` | Token 过期可配置；封禁后每次请求均检查，无需等待 Token 过期 |
| 消息归属权校验 | `chat_handler.go` | 编辑消息时服务端强制校验归属，防越权 |
| 初始密码环境变量化 | `user_repository.go` | 从 `ADMIN_INITIAL_PASSWORD` 读取，避免硬编码 |
| 密码 bcrypt 哈希 | `user_repository.go` | 密码不可逆存储，bcrypt 错误不再被忽略 |
| Redis 滑动窗口限流 | `rate_limit.go` | 防止暴力破解和 DDoS |
| 管理员权限控制 | `admin.go` | 基于角色的访问控制 |
| Tunnel 接口鉴权 | `admin_handler.go` | 公网地址查询接口移入管理员路由，普通用户不可访问 |

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

- **256 哈希分桶客户端池**：按用户 ID 的 FNV-32a 哈希分配桶，每桶独立 goroutine，降低锁竞争；`DisconnectUser`/`UpdateUsername` 通过同一哈希直接定位目标桶，无需全量扫描
- **64 分片频道锁**：按频道 ID 的 FNV-32a 哈希分配分片，支持频道操作并发
- **同步持久化**：消息先写库获取数据库 ID，再向频道广播；保证广播消息携带有效 ID，客户端可精准追踪消息

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

| 字段              | 类型           | 说明               |
| --------------- | ------------ | ---------------- |
| `id`            | varchar(36)  | UUID 主键          |
| `username`      | varchar(64)  | 唯一，字母数字下划线       |
| `password_hash` | varchar(255) | bcrypt 哈希        |
| `role`          | varchar(16)  | `admin` 或 `user` |
| `banned`        | boolean      | 封禁标志             |

**频道（Channel）**

| 字段           | 类型          | 说明       |
| ------------ | ----------- | -------- |
| `id`         | varchar(36) | UUID 主键  |
| `name`       | varchar(64) | 唯一频道名    |
| `created_by` | varchar(36) | 创建者用户 ID |
| `created_at` | timestamp   | 创建时间     |

**消息（Message）**

| 字段           | 类型          | 说明          |
| ------------ | ----------- | ----------- |
| `id`         | varchar(36) | UUID 主键     |
| `channel_id` | varchar(36) | 索引外键，关联频道   |
| `user_id`    | varchar(36) | 作者用户 ID     |
| `username`   | varchar(64) | 作者用户名（冗余字段） |
| `content`    | text        | 消息内容        |
| `created_at` | timestamp   | 索引创建时间      |

### 中间件链

| 中间件                     | 作用                                                           |
| ----------------------- | ------------------------------------------------------------ |
| `gin.Recovery()`        | 捕获 panic，返回 500                                              |
| `gzip.Gzip()`           | 响应压缩（默认级别）                                                   |
| `OtelMiddleware()`      | 为每个请求创建 OpenTelemetry Span                                   |
| `LoggingMiddleware()`   | 记录请求方法、路径、状态码、耗时、IP、User-Agent                                                      |
| `AuthMiddleware()`      | 验证 JWT Bearer Token + 实时查库检查封禁状态，将 `user_id`、`username`、`role` 存入 Context |
| `AdminMiddleware()`     | 校验管理员角色（旧 Token 无 role 时回退查库）                                                      |
| `RateLimitMiddleware()` | Redis 滑动窗口限流（按客户端 IP）                                                               |

---

## Makefile 命令

| 命令               | 说明                                           |
| ---------------- | -------------------------------------------- |
| `make build`     | 编译三个二进制文件（api、grpc、ws）到 `bin/`               |
| `make run`       | 直接运行 API 服务器                                 |
| `make start`     | 运行 API 服务器 (`go run ./cmd/api/main.go`)      |
| `make run-grpc`  | 运行独立 gRPC 服务器                                |
| `make run-ws`    | 运行独立 WebSocket 服务器                           |
| `make test`      | 运行所有单元测试（`go test ./... -v`）                |
| `make bench`     | 运行内置 HTTP 压测工具                               |
| `make hotreload` | 使用文件监听热重载运行                                  |
| `make proto`     | 重新生成 Protobuf Go 代码                          |
| `make clean`     | 删除 `bin/` 目录和生成的 Protobuf 文件                 |

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

## 变更日志

### 2026-07-31 架构重构、消息总线多路复用与全套单元测试

**核心架构重构（P0）**

- ⚡ **Redis 消息总线多路复用**：`pkg/redisbus` 彻底重构为**单个 `redis.PubSub` 物理连接 + 本地引用计数器 (`channelRefs`)** 模式。解决此前“每订阅一个频道新建一个 PubSub 连接”导致的并发连接爆炸问题，大幅提升多频道扩展能力。
- 🛡️ **WebSocket 消息写库超时防护**：WebSocket 消息写库过程增加 3 秒超时监控，替换 `time.After` 为 `time.NewTimer` + `defer timer.Stop()` 避免内存泄露，彻底防范数据库慢查询阻塞 WebSocket 读写循环。
- 🔒 **并发安全与 Data Race 修复**：为 `redisbus.MessageBus` 的回调句柄加上互斥锁保护；`pkg/ws` 广播广播过程收敛至线程安全的 `SendMessage` 方法，避免连接关闭时的 channel 竞争。

**质量与基础设施（P1）**

- 🧪 **引入全套自动化单元测试**：覆盖 `cmd/`, `internal/`（handler, repository, middleware, database, logger, tunnel, config）, `pkg/`（jwt, limiter, redisbus, ws, otel, grpcclient）全部关键路径，单测通行率 100%。
- 🌐 **原生 Golang Cloudflare 隧道引擎**：`internal/tunnel` 实现基于 `exec.CommandContext` 的原生 Golang 进程与日志管道解析引擎，替代原外部脚本。
- 🧹 **脚本与环境清理**：移除冗余的 Windows 批处理及 PowerShell 启动脚本，统一收口至 `Makefile`；配置隔离 `.env`，防个人敏感秘钥泄漏至公有 GitHub 仓库。

### 2026-07-03 安全加固 & 架构修复

**安全修复（P0）**

- 🔒 **修复越权编辑漏洞**：`EditMessage` 增加 `msg.UserID == userID` 服务端校验，任何用户无法编辑他人消息
- 🔒 **封禁立即生效**：`AuthMiddleware` 签名更新为 `AuthMiddleware(cfg, userRepo)`，每次请求实时查库检查封禁状态，不再依赖 JWT 自然过期
- 🔒 **管理员初始密码安全化**：移除硬编码 `admin123`，改为从 `ADMIN_INITIAL_PASSWORD` 环境变量读取；bcrypt 错误不再被 `_` 忽略
- 🔒 **WebSocket Upgrader CheckOrigin 统一**：Upgrader 设为 `always true`，Origin 检查统一由 `checkOrigin()` 函数单点把守，消除双重逻辑冲突
- 🔒 **Tunnel 接口移入管理员路由**：`GET /api/tunnel` → `GET /api/admin/tunnel`，普通用户不再能查询公网地址

**数据一致性修复（P1）**

- 🛠 **`SetUsername` 事务化**：用户表更新与历史消息用户名同步现在在同一个数据库事务中执行，防止部分失败导致数据不一致
- 🛠 **移除死代码**：删除从未写入的 `persistCh` channel 和相关 `persistWorker` goroutine（启动了 CPU 核数个 goroutine 但 channel 永远为空）

**性能优化（P2）**

- ⚡ **`UserRepository` 新增 `Count()` 接口**：`GET /api/admin/stats` 不再全量加载用户列表，改用 `COUNT(*)` 查询
- ⚡ **`DisconnectUser`/`UpdateUsername` 精准定位**：利用 FNV-32a 哈希直接定位目标桶，从遍历全部 256 个桶降至仅操作 1 个桶

**代码质量（P3）**

- 🧹 移除 `DeleteMyMessage` 中残留的 5 条 `[DEBUG]` 日志（生产环境泄露用户 ID）
- 🧹 `GetMessages` 的 `limit` 参数解析错误不再被 `_` 忽略，错误时回退到默认值 50
- 🧹 新增 `.gitattributes`，统一 LF 换行符策略（`.bat`/`.ps1` 保留 CRLF）

---

## 开源协议

无
