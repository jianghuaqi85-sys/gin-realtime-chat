@echo off
chcp 65001 >nul 2>&1
echo ========================================
echo   更新隧道 URL 工具
echo ========================================
echo.
echo 使用说明:
echo   1. 查看 CloudflareTunnel 窗口
echo   2. 复制显示的 https://xxx.trycloudflare.com 地址
echo   3. 粘贴到下方
echo.
echo ========================================
echo.

cd /d "%~dp0"

set /p "NEW_URL=请输入新的隧道 URL: "

if "%NEW_URL%"=="" (
    echo 错误: URL 不能为空
    pause
    exit /b 1
)

:: 验证 URL 格式
echo %NEW_URL% | findstr /i "trycloudflare.com" >nul
if errorlevel 1 (
    echo 警告: URL 不包含 trycloudflare.com，请确认是否正确
    echo.
    set /p "CONFIRM=是否继续？(Y/N): "
    if /i not "%CONFIRM%"=="Y" (
        echo 已取消
        pause
        exit /b 0
    )
)

:: 保存到文件
echo %NEW_URL% > .tunnel_url

echo.
echo ========================================
echo   URL 已更新！
echo   新地址: %NEW_URL%
echo   文件位置: .tunnel_url
echo ========================================
echo.
pause
