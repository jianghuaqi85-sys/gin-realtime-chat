@echo off
chcp 65001 >nul 2>&1
echo ========================================
echo   聊天室服务器启动脚本
echo ========================================
echo.

:: 配置（按需修改）
set CLOUDFLARED=D:\cloudflared.exe
set REDIS_SERVER=D:\Redis\redis-server.exe
set REDIS_CONFIG=D:\Redis\redis.windows.conf

:: 启动 Redis（如果未运行）
tasklist /fi "imagename eq redis-server.exe" 2>nul | find /i "redis-server.exe" >nul
if errorlevel 1 (
    echo [1/3] 启动 Redis...
    start /b "" "%REDIS_SERVER%" "%REDIS_CONFIG%"
    timeout /t 2 /nobreak >nul
) else (
    echo [1/3] Redis 已在运行
)

:: 启动 Go 服务器
echo [2/3] 启动 API 服务器...
start "ChatServer" cmd /c "cd /d %~dp0 && go run ./cmd/api/"

:: 等待服务器启动
timeout /t 3 /nobreak >nul

:: 启动 Cloudflare Tunnel
echo [3/3] 启动 Cloudflare Tunnel...
start "CloudflareTunnel" cmd /c "%CLOUDFLARED% tunnel --url http://localhost:8080"

echo.
echo ========================================
echo   服务器启动完成！
echo   本地访问: http://localhost:8080
echo   公网地址: 请查看 CloudflareTunnel 窗口
echo ========================================
echo.
pause
