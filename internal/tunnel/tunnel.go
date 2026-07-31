package tunnel

import (
	"bufio"
	"context"
	"errors"
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
	cloudflaredPath string
	port            string
	urlFile         string

	cmd    *exec.Cmd
	cancel context.CancelFunc // 用于优雅取消进程
	url    string
	mu     sync.RWMutex
}

// NewManager 创建隧道管理器
func NewManager(cloudflaredPath, port, urlFile string) *Manager {
	return &Manager{
		cloudflaredPath: cloudflaredPath,
		port:            port,
		urlFile:         urlFile,
	}
}

// Start 启动 Cloudflare Tunnel 并阻塞等待 URL 就绪
func (m *Manager) Start(ctxs ...context.Context) error {
	ctx := context.Background()
	if len(ctxs) > 0 && ctxs[0] != nil {
		ctx = ctxs[0]
	}

	m.mu.Lock()
	if m.cmd != nil {
		m.mu.Unlock()
		return errors.New("tunnel 已经启动")
	}

	cmdCtx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.cmd = exec.CommandContext(cmdCtx, m.cloudflaredPath, "tunnel", "--url", "http://localhost:"+m.port)
	m.mu.Unlock()

	stderr, err := m.cmd.StderrPipe()
	if err != nil {
		m.Stop()
		return fmt.Errorf("创建 stderr pipe 失败: %w", err)
	}

	if err := m.cmd.Start(); err != nil {
		m.Stop()
		return fmt.Errorf("启动 cloudflared 失败: %w", err)
	}

	urlReady := make(chan struct{})
	processExit := make(chan error, 1)

	// 1. 扫描日志的 Goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		var once sync.Once
		for scanner.Scan() {
			line := scanner.Text()

			if match := urlRegex.FindString(line); match != "" {
				once.Do(func() {
					m.mu.Lock()
					m.url = match
					m.mu.Unlock()

					if err := os.WriteFile(m.urlFile, []byte(match), 0644); err != nil {
						log.Printf("[WARN] 保存隧道 URL 失败: %v", err)
					} else {
						log.Printf("Cloudflare Tunnel 地址就绪: %s", match)
					}
					close(urlReady)
				})
			}

			if strings.Contains(line, "ERR") || strings.Contains(line, "error") {
				log.Printf("[cloudflared] %s", line)
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("[WARN] 读取 cloudflared 日志中断: %v", err)
		}
	}()

	// 2. 监控进程退出的 Goroutine
	go func() {
		err := m.cmd.Wait()
		processExit <- err
	}()

	// 3. 阻塞等待：URL 就绪 / 进程崩溃 / 超时
	select {
	case <-urlReady:
		return nil
	case err := <-processExit:
		m.Stop()
		if err != nil {
			return fmt.Errorf("cloudflared 进程异常退出: %w", err)
		}
		return errors.New("cloudflared 退出且未生成 URL")
	case <-ctx.Done():
		m.Stop()
		return fmt.Errorf("等待隧道启动超时或被取消: %w", ctx.Err())
	}
}

// GetURL 获取当前隧道 URL
func (m *Manager) GetURL() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.url
}

// Stop 优雅停止隧道
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
	m.cmd = nil
	m.url = ""

	_ = os.Remove(m.urlFile)
}
