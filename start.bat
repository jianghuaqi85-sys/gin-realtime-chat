@echo off
chcp 65001 >nul 2>&1
echo ========================================
echo   聊天室服务器启动脚本
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
echo 等待隧道建立...
timeout /t 5 /nobreak >nul

:: 获取新的隧道 URL
echo [3/3] 获取公网地址...
set "TUNNEL_URL="

:: 使用 PowerShell 从 cloudflared 进程捕获输出
for /f "tokens=*" %%i in ('powershell -Command "Get-Content 'C:\Users\86198\Desktop\GIn\.tunnel_url' -ErrorAction SilentlyContinue"') do (
    set "TUNNEL_URL=%%i"
)

if "%TUNNEL_URL%"=="" (
    echo [警告] 无法自动获取隧道 URL
    echo 请查看 CloudflareTunnel 窗口获取公网地址
    echo.
    echo ========================================
    echo   服务器启动完成！
    echo   本地访问: http://localhost:8080
    echo   公网地址: 请查看 CloudflareTunnel 窗口
    echo ========================================
) else (
    echo.
    echo ========================================
    echo   服务器启动完成！
    echo   本地访问: http://localhost:8080
    echo   公网地址: %TUNNEL_URL%
    echo.
    echo   提示: 复制上方地址分享给用户
    echo ========================================
)
echo.
pause