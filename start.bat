@echo off
chcp 65001 >nul 2>&1
echo ========================================
echo   聊天室服务器启动脚本
echo ========================================
echo.

cd /d "%~dp0"

:: 启动 Go 服务器
echo [1/2] 启动 API 服务器...
start "ChatServer" cmd /k "go run ./cmd/api"

:: 等待服务器启动
echo 等待服务器启动...
timeout /t 5 /nobreak >nul

:: 启动 Cloudflare Tunnel
echo [2/2] 启动 Cloudflare Tunnel...
start "CloudflareTunnel" cmd /k "D:\cloudflared.exe tunnel --url http://localhost:8080"

echo.
echo ========================================
echo   服务器启动完成！
echo   本地访问: http://localhost:8080
echo   公网地址: 请查看 CloudflareTunnel 窗口
echo ========================================
echo.
pause