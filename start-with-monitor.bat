@echo off
chcp 65001 >nul 2>&1
echo ========================================
echo   聊天室服务器启动脚本 (带监控)
echo ========================================
echo.

cd /d "%~dp0"

:: 停止旧的 cloudflared 进程
echo [0/3] 清理旧进程...
taskkill /IM cloudflared.exe /F >nul 2>&1
timeout /t 1 /nobreak >nul

:: 启动 Go 服务器
echo [1/3] 启动 API 服务器...
start "ChatServer" cmd /k "go run ./cmd/api"

:: 等待服务器启动
echo 等待服务器启动...
timeout /t 5 /nobreak >nul

:: 启动 Cloudflare Tunnel
echo [2/3] 启动 Cloudflare Tunnel...
start "CloudflareTunnel" cmd /k "D:\cloudflared.exe tunnel --url http://localhost:8080"

:: 等待隧道建立
echo [3/3] 等待隧道建立...
timeout /t 8 /nobreak >nul

:: 尝试从 cloudflared 窗口获取 URL (使用 PowerShell)
echo 正在尝试获取隧道 URL...
for /f "tokens=*" %%i in ('powershell -Command "Get-Content -Path '$env:TEMP\cloudflared*.log' -Tail 20 -ErrorAction SilentlyContinue | Select-String -Pattern 'trycloudflare.com' | Select-Object -Last 1"') do (
    for /f "tokens=2 delims= " %%j in ("%%i") do (
        echo %%j > .tunnel_url
        echo.
        echo ========================================
        echo   服务器启动完成！
        echo   本地访问: http://localhost:8080
        echo   公网地址: %%j
        echo.
        echo   提示: 复制上方地址分享给用户
        echo ========================================
        pause
        exit /b 0
    )
)

:: 如果无法自动获取，提示用户手动输入
echo.
echo ========================================
echo   提示: 无法自动获取隧道 URL
echo ========================================
echo.
echo 请查看 CloudflareTunnel 窗口，找到类似以下格式的地址:
echo   https://xxxxx-xxxxx.trycloudflare.com
echo.
echo 然后运行 update-tunnel-url.bat 手动更新
echo.
echo ========================================
echo   服务器启动完成！
echo   本地访问: http://localhost:8080
echo   公网地址: 请查看 CloudflareTunnel 窗口
echo ========================================
echo.
pause
