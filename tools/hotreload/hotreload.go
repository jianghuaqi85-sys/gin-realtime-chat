package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/fsnotify/fsnotify"
)

const tempBinName = "./tmp-api-server"

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("Error creating watcher:", err)
		return
	}
	defer watcher.Close()

	// 1. 优化：正确使用 filepath.SkipDir 阻止深度遍历，大幅提升启动速度
	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if isExcluded(info.Name()) {
				return filepath.SkipDir // 告诉 Walk 函数不要进入该目录
			}
			return watcher.Add(path)
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error walking directory:", err)
		return
	}

	var cmd *exec.Cmd

	// 2. 优化：利用 time.Timer 实现防抖 (Debouncing)
	debounceTimer := time.NewTimer(0)
	<-debounceTimer.C

	// 启动文件监听循环
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				// 监听修改、重命名、创建事件
				if (event.Has(fsnotify.Write) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Create)) && isGoFile(event.Name) {
					// 每次检测到变更，重置计时器。500ms 内的多次变更会被合并
					debounceTimer.Reset(500 * time.Millisecond)
				}
			case err := <-watcher.Errors:
				fmt.Println("Watcher error:", err)
			}
		}
	}()

	// 手动触发第一次启动
	debounceTimer.Reset(100 * time.Millisecond)

	// 核心重启循环
	for range debounceTimer.C {
		fmt.Println("\n[Live-Reload] Detected changes, restarting...")

		// 3. 优化：确保彻底杀死旧进程并等待操作系统释放端口
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait() // 必须等待旧进程彻底死亡
		}

		// 4. 优化：先编译，再运行。告别 go run 带来的孤儿进程问题
		fmt.Println("[Live-Reload] Building...")
		buildCmd := exec.Command("go", "build", "-o", getBinName(), "./cmd/api")
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr

		if err := buildCmd.Run(); err != nil {
			fmt.Println("[Live-Reload] Build failed, waiting for next change...")
			continue // 编译失败时不启动服务，等待下一次修改
		}

		// 启动编译好的二进制文件
		cmd = exec.Command(getBinName())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		fmt.Println("[Live-Reload] Starting server...")
		if err := cmd.Start(); err != nil {
			fmt.Println("[Live-Reload] Error starting server:", err)
		}
	}
}

// 检查是否为需要排除的目录（只匹配目录名本身即可）
func isExcluded(dirName string) bool {
	excluded := []string{"node_modules", ".git", "vendor", "dist", "build", "tmp", "assets"}
	for _, dir := range excluded {
		if dirName == dir {
			return true
		}
	}
	return false
}

func isGoFile(path string) bool {
	return filepath.Ext(path) == ".go"
}

// 处理跨平台二进制后缀
func getBinName() string {
	if runtime.GOOS == "windows" {
		return tempBinName + ".exe"
	}
	return tempBinName
}
