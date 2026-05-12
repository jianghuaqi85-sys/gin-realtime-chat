@echo off
chcp 65001 >nul 2>&1
echo ========================================
echo   聊天室服务器启动脚本 (智能版)
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

:: 启动 Cloudflare Tunnel 并捕获 URL
echo [2/3] 启动 Cloudflare Tunnel...
echo 正在等待隧道建立（最多 30 秒）...
echo.

:: 使用 PowerShell 启动 cloudflared 并捕获输出
powershell -ExecutionPolicy Bypass -File "%~dp0launch-tunnel.ps1"

:: 读取保存的 URL
set "TUNNEL_URL="
if exist "%~dp0.tunnel_url" (
    set /p TUNNEL_URL=<"%~dp0.tunnel_url"
)

echo.
echo ========================================
echo   服务器启动完成！
echo   本地访问: http://localhost:8080
if "%TUNNEL_URL%"=="" (
    echo   公网地址: 请查看 CloudflareTunnel 窗口
) else (
    echo   公网地址: %TUNNEL_URL%
    echo.
    echo   提示: 复制上方地址分享给用户
)
echo ========================================
echo.
pause
