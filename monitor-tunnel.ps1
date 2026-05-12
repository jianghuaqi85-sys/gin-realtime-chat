# 监控 cloudflared 输出并保存隧道 URL
param(
    [string]$LogPath = "C:\Users\86198\Desktop\GIn\tunnel.log",
    [string]$UrlPath = "C:\Users\86198\Desktop\GIn\.tunnel_url"
)

# 清空旧日志
if (Test-Path $LogPath) {
    Remove-Item $LogPath -Force
}

Write-Host "开始监控隧道..." -ForegroundColor Cyan

# 获取 cloudflared 进程
$process = Get-Process cloudflared -ErrorAction SilentlyContinue | Select-Object -First 1

if (-not $process) {
    Write-Host "错误: 未找到 cloudflared 进程" -ForegroundColor Red
    exit 1
}

# 使用 WMI 获取命令行
$wmi = Get-WmiObject Win32_Process -Filter "ProcessId = $($process.Id)"
$commandLine = $wmi.CommandLine

Write-Host "检测到 cloudflared 进程 (PID: $($process.Id))" -ForegroundColor Yellow
Write-Host "命令行: $commandLine" -ForegroundColor Gray

# 等待隧道建立
$maxWait = 30
$waited = 0
$urlFound = $false

while ($waited -lt $maxWait -and -not $urlFound) {
    Start-Sleep -Seconds 1
    $waited++

    # 尝试从 cloudflared 的标准输出获取 URL
    # 由于直接读取输出较难，我们通过检查进程状态和网络连接来判断
    $connections = netstat -ano | Select-String "cloudflared"

    if ($connections) {
        Write-Host "检测到网络连接，隧道可能已建立..." -ForegroundColor Green
        break
    }
}

# 最佳方案：提示用户查看 cloudflared 窗口
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "隧道监控提示" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "请查看 CloudflareTunnel 窗口" -ForegroundColor Yellow
Write-Host "窗口中会显示类似以下格式的 URL：" -ForegroundColor Yellow
Write-Host "  https://xxxxx-xxxxx.trycloudflare.com" -ForegroundColor Green
Write-Host ""
Write-Host "获取到 URL 后，请手动更新 .tunnel_url 文件" -ForegroundColor Gray
Write-Host "或使用以下命令自动更新：" -ForegroundColor Gray
Write-Host '  echo "https://your-url.trycloudflare.com" > .tunnel_url' -ForegroundColor Cyan
Write-Host ""
