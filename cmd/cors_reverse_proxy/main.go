package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"cors_reverse_proxy/internal/config"
	"cors_reverse_proxy/internal/server"
)

func main() {
	appDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}
	fmt.Println("Current working directory:", appDir)

	server.WriteKillScript(appDir)

	cfg, err := config.Load(filepath.Join(appDir, "config.json5"))
	if err != nil {
		log.Printf("加载配置失败，使用默认值: %v\n", err)
	}

	if err := server.Run(cfg); err != nil {
		log.Fatalln(err)
	}
}
