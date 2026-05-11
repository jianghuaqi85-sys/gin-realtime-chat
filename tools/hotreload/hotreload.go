package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

func main() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("Error creating watcher:", err)
		return
	}
	defer watcher.Close()

	err = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && !isExcluded(path) {
			err = watcher.Add(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error walking directory:", err)
		return
	}

	var cmd *exec.Cmd
	restart := make(chan bool, 1)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Has(fsnotify.Write) && isGoFile(event.Name) {
					restart <- true
				}
			case err := <-watcher.Errors:
				fmt.Println("Watcher error:", err)
			}
		}
	}()

	restart <- true

	for range restart {
		if cmd != nil {
			cmd.Process.Kill()
		}

		cmd = exec.Command("go", "run", "./cmd/api/main.go")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		fmt.Println("Starting server...")
		err := cmd.Start()
		if err != nil {
			fmt.Println("Error starting server:", err)
			time.Sleep(1 * time.Second)
			restart <- true
			continue
		}

		go func() {
			cmd.Wait()
		}()

		time.Sleep(500 * time.Millisecond)
	}
}

func isExcluded(path string) bool {
	excluded := []string{"node_modules", ".git", "vendor", "dist", "build"}
	for _, dir := range excluded {
		if filepath.Base(path) == dir {
			return true
		}
	}
	return false
}

func isGoFile(path string) bool {
	return filepath.Ext(path) == ".go"
}
