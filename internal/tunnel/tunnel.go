package tunnel

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

var urlRegex = regexp.MustCompile(`https://[a-zA-Z0-9\-]+\.trycloudflare\.com`)

type Manager struct {
	cmd     *exec.Cmd
	url     string
	mu      sync.RWMutex
	urlFile string
}

// NewManager 创建隧道管理器
// cloudflaredPath: cloudflared 可执行文件路径
// port: 本地服务端口
// urlFile: 保存 URL 的文件路径
func NewManager(cloudflaredPath, port, urlFile string) *Manager {
	return &Manager{
		cmd:     exec.Command(cloudflaredPath, "tunnel", "--url", "http://localhost:"+port),
		urlFile: urlFile,
	}
}

// Start 启动 Cloudflare Tunnel 并捕获 URL
func (m *Manager) Start() error {
	stderr, err := m.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建 stderr pipe 失败: %w", err)
	}

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("启动 cloudflared 失败: %w", err)
	}

	// 从 stderr 读取输出，捕获 URL
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()

			// 检测 URL
			if match := urlRegex.FindString(line); match != "" {
				m.mu.Lock()
				m.url = match
				m.mu.Unlock()

				// 保存到文件
				if err := os.WriteFile(m.urlFile, []byte(match), 0644); err != nil {
					log.Printf("[WARN] 保存隧道 URL 失败: %v", err)
				} else {
					log.Printf("Cloudflare Tunnel 地址: %s", match)
				}
			}

			// 输出 cloudflared 日志
			if strings.Contains(line, "ERR") || strings.Contains(line, "error") {
				log.Printf("[cloudflared] %s", line)
			}
		}
	}()

	// 等待进程结束
	go func() {
		if err := m.cmd.Wait(); err != nil {
			log.Printf("[WARN] cloudflared 进程退出: %v", err)
		}
	}()

	return nil
}

// GetURL 获取当前隧道 URL
func (m *Manager) GetURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.url
}

// Stop 停止隧道
func (m *Manager) Stop() {
	if m.cmd != nil && m.cmd.Process != nil {
		m.cmd.Process.Kill()
	}
	// 清理 URL 文件
	os.Remove(m.urlFile)
}
