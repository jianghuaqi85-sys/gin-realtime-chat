# 自动监控隧道 URL 变化并更新 .tunnel_url 文件
param(
    [string]$TunnelUrlPath = "C:\Users\86198\Desktop\GIn\.tunnel_url",
    [int]$CheckIntervalSeconds = 5
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "隧道 URL 自动监控器" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "此脚本会持续监控 cloudflared 输出" -ForegroundColor Gray
Write-Host "当检测到新的隧道 URL 时会自动更新" -ForegroundColor Gray
Write-Host ""
Write-Host "按 Ctrl+C 停止监控" -ForegroundColor Yellow
Write-Host ""

# 获取当前 URL（如果有）
$currentUrl = ""
if (Test-Path $TunnelUrlPath) {
    $currentUrl = Get-Content $TunnelUrlPath -ErrorAction SilentlyContinue
    if ($currentUrl) {
        Write-Host "当前 URL: $currentUrl" -ForegroundColor Green
    }
}

$lastUrl = $currentUrl

# 主监控循环
while ($true) {
    Start-Sleep -Seconds $CheckIntervalSeconds

    # 获取 cloudflared 进程
    $process = Get-Process cloudflared -ErrorAction SilentlyContinue | Select-Object -First 1

    if ($process) {
        # 尝试从标准错误读取（cloudflared 通常在这里输出）
        try {
            $stderr = $process.StandardError
            if ($stderr) {
                $line = $stderr.ReadLine()
                if ($line -and $line -match "https://[a-zA-Z0-9-]+\.trycloudflare\.com") {
                    $newUrl = $Matches[0]

                    if ($newUrl -ne $lastUrl) {
                        Write-Host ""
                        Write-Host "[$(Get-Date -Format 'HH:mm:ss')] 检测到新 URL！" -ForegroundColor Green
                        Write-Host "新地址: $newUrl" -ForegroundColor Cyan

                        # 更新文件
                        $newUrl | Out-File -FilePath $TunnelUrlPath -Encoding UTF8 -NoNewline
                        Write-Host "已更新 .tunnel_url 文件" -ForegroundColor Gray

                        $lastUrl = $newUrl
                    }
                }
            }
        } catch {
            # 忽略读取错误
        }
    } else {
        Write-Host "[$(Get-Date -Format 'HH:mm:ss')] cloudflared 未运行" -ForegroundColor Yellow
    }
}
