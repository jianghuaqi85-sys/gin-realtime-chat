# 隧道 URL 自动检测方案

## 问题

免费的 Cloudflare 临时隧道每次重启都会生成新的随机 URL，导致需要手动更新地址。

## 解决方案

提供了多种脚本帮助自动管理和更新隧道 URL。

## 文件说明

### 核心文件
- `.tunnel_url` - 存储当前的隧道 URL
- `start.bat` - 原始启动脚本（已更新，会尝试读取 URL）
- `start-smart.bat` - 智能启动脚本（自动捕获 URL）
- `update-tunnel-url.bat` - 手动更新 URL 工具

### 辅助脚本
- `launch-tunnel.ps1` - 智能启动 cloudflared 的 PowerShell 脚本
- `auto-update-url.ps1` - 持续监控并自动更新 URL 的后台脚本
- `monitor-tunnel.ps1` - 简单的隧道监控脚本

## 使用方法

### 方法 1: 手动更新（推荐，最简单）

1. 运行 `start.bat` 启动服务
2. 查看 CloudflareTunnel 窗口，找到公网地址
3. 双击运行 `update-tunnel-url.bat`
4. 粘贴 URL 并回车

```batch
# 例如:
# https://artists-politicians-struck-vocal.trycloudflare.com
```

### 方法 2: 使用智能启动脚本

```batch
start-smart.bat
```

这个脚本会尝试自动捕获新的 URL，如果成功会自动显示。

### 方法 3: 后台自动监控

打开两个命令行窗口：

**窗口 1 - 启动服务：**
```batch
start.bat
```

**窗口 2 - 启动监控：**
```powershell
powershell -ExecutionPolicy Bypass -File auto-update-url.ps1
```

监控脚本会持续运行，当检测到新 URL 时自动更新 `.tunnel_url` 文件。

## 配置说明

如果需要修改配置，编辑 `launch-tunnel.ps1`：

```powershell
# cloudflared 路径
$CloudflaredPath = "D:\cloudflared.exe"

# 本地服务地址
$LocalUrl = "http://localhost:8080"

# URL 保存路径
$TunnelUrlPath = "C:\Users\86198\Desktop\GIn\.tunnel_url"

# 等待超时时间（秒）
$TimeoutSeconds = 30
```

## 常见问题

### Q: 为什么不能完全自动？
A: cloudflared 的输出捕获在 Windows 上比较复杂，且临时隧道的 URL 是随机生成的，无法预测。

### Q: URL 什么时候会变？
A: 以下情况 URL 会变化：
- 重启 cloudflared 进程
- 电脑重启
- cloudflared 崩溃后重启

### Q: 如何获得固定的 URL？
A: 使用 Cloudflare 账号创建命名隧道（Named Tunnel），可以绑定自定义域名。

### Q: 监控脚本没有检测到 URL？
A: 这是正常现象，因为 PowerShell 难以直接读取其他进程的输出。建议使用手动更新方法。

## 快速开始

```batch
# 1. 启动服务
start.bat

# 2. 查看 CloudflareTunnel 窗口获取地址

# 3. 更新 URL（选择一种方式）
# 方式 A: 双击 update-tunnel-url.bat 手动输入
# 方式 B: 直接编辑 .tunnel_url 文件

# 4. 验证
# 打开 .tunnel_url 文件确认地址正确
```

## 注意事项

- 免费隧道没有 SLA 保证，不建议用于生产环境
- 临时 URL 会在进程重启时变化
- 如需稳定 URL，请使用 Cloudflare 账号创建命名隧道
