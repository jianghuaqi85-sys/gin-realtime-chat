@echo off
:: 启动 Cloudflare Tunnel 并保存 URL 到 .tunnel_url 文件
set CLOUDFLARED=D:\cloudflared.exe

echo 启动 Cloudflare Tunnel...
:: 启动 cloudflared，捕获输出中的 URL
for /f "tokens=*" %%i in ('%CLOUDFLARED% tunnel --url http://localhost:8080 2^>^&1 ^| findstr /i "trycloudflare.com"') do (
    echo %%i
    :: 提取 https:// 部分
    for /f "tokens=2 delims=|" %%j in ("%%i") do (
        set "URL=%%j"
        :: 去掉前后空格
        for /f "tokens=*" %%k in ("%%j") do (
            echo %%k > "%~dp0.tunnel_url"
            echo.
            echo 公网地址已保存: %%k
        )
    )
)
:: 保持窗口打开
%CLOUDFLARED% tunnel --url http://localhost:8080
