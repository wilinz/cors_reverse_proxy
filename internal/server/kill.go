package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

func killHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "程序即将退出"})
	go func() {
		time.Sleep(1500 * time.Millisecond)
		os.Exit(0)
	}()
}

// WriteKillScript writes a platform-specific stop script next to the binary.
func WriteKillScript(appDir string) {
	pid := os.Getpid()
	if runtime.GOOS == "windows" {
		content := fmt.Sprintf("@echo off\r\ntaskkill /PID %d /F 2>nul\r\necho Service stopped\r\n", pid)
		path := filepath.Join(appDir, "kill.bat")
		if err := os.WriteFile(path, []byte(content), 0755); err != nil {
			log.Printf("生成 kill.bat 失败: %v\n", err)
		} else {
			fmt.Println("已生成 kill.bat")
		}
		return
	}
	content := fmt.Sprintf("#!/bin/sh\nkill %d 2>/dev/null\necho 'Service stopped'\n", pid)
	path := filepath.Join(appDir, "kill.sh")
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		log.Printf("生成 kill.sh 失败: %v\n", err)
	} else {
		fmt.Println("已生成 kill.sh")
	}
}
