# 智能启动 cloudflared 并捕获隧道 URL
param(
    [string]$CloudflaredPath = "D:\cloudflared.exe",
    [string]$LocalUrl = "http://localhost:8080",
    [string]$TunnelUrlPath = "C:\Users\86198\Desktop\GIn\.tunnel_url",
    [int]$TimeoutSeconds = 30
)

Write-Host "正在启动 Cloudflare Tunnel..." -ForegroundColor Cyan
Write-Host "本地地址: $LocalUrl" -ForegroundColor Gray
Write-Host ""

# 清空旧的 URL 文件
if (Test-Path $TunnelUrlPath) {
    Remove-Item $TunnelUrlPath -Force
}

# 启动 cloudflared 进程
Write-Host "启动 cloudflared..." -ForegroundColor Yellow

$process = Start-Process -FilePath $CloudflaredPath `
    -ArgumentList "tunnel --url $LocalUrl" `
    -PassThru `
    -RedirectStandardOutput "$env:TEMP\cloudflared_stdout.log" `
    -RedirectStandardError "$env:TEMP\cloudflared_stderr.log" `
    -WindowStyle Hidden

Write-Host "cloudflared 已启动 (PID: $($process.Id))" -ForegroundColor Gray

# 等待并监控输出
$startTime = Get-Date
$urlFound = $false

Write-Host "等待隧道建立..." -ForegroundColor Yellow

while (-not $urlFound -and ((Get-Date) - $startTime).TotalSeconds -lt $TimeoutSeconds) {
    Start-Sleep -Milliseconds 500

    # 检查标准输出日志
    if (Test-Path "$env:TEMP\cloudflared_stdout.log") {
        $stdoutContent = Get-Content "$env:TEMP\cloudflared_stdout.log" -ErrorAction SilentlyContinue
        foreach ($line in $stdoutContent) {
            if ($line -match "https://[a-zA-Z0-9-]+\.trycloudflare\.com") {
                $urlFound = $true
                $tunnelUrl = $Matches[0]
                break
            }
        }
    }

    # 检查标准错误日志
    if (-not $urlFound -and Test-Path "$env:TEMP\cloudflared_stderr.log") {
        $stderrContent = Get-Content "$env:TEMP\cloudflared_stderr.log" -ErrorAction SilentlyContinue
        foreach ($line in $stderrContent) {
            if ($line -match "https://[a-zA-Z0-9-]+\.trycloudflare\.com") {
                $urlFound = $true
                $tunnelUrl = $Matches[0]
                break
            }
        }
    }

    # 显示进度
    $elapsed = [math]::Round(((Get-Date) - $startTime).TotalSeconds)
    Write-Host "`r已等待 $elapsed 秒..." -NoNewline -ForegroundColor Gray
}

Write-Host ""  # 换行

if ($urlFound) {
    Write-Host "隧道已建立！" -ForegroundColor Green
    Write-Host "公网地址: $tunnelUrl" -ForegroundColor Cyan

    # 保存 URL 到文件
    $tunnelUrl | Out-File -FilePath $TunnelUrlPath -Encoding UTF8 -NoNewline
    Write-Host "URL 已保存到: $TunnelUrlPath" -ForegroundColor Gray
} else {
    Write-Host "警告: 在 $TimeoutSeconds 秒内未检测到 URL" -ForegroundColor Yellow
    Write-Host "请手动查看 cloudflared 窗口获取地址" -ForegroundColor Yellow
}
