package config

import (
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/titanous/json5"
)

type Config struct {
	Listening string `json:"listening"`
	Token     string `json:"token"`
	HttpProxy string `json:"http_proxy"`
	SkipTLS   bool   `json:"skip_tls"`
}

// Build-time defaults (override with -ldflags "-X cors_reverse_proxy/internal/config.DefaultXxx=...").
var (
	DefaultListening = "0.0.0.0:10010"
	DefaultToken     = ""
	DefaultHttpProxy = ""
	DefaultSkipTLS   = "true"
)

func Default() Config {
	token := DefaultToken
	if token == "" {
		token = uuid.NewString()
	}
	return Config{
		Listening: DefaultListening,
		Token:     token,
		HttpProxy: DefaultHttpProxy,
		SkipTLS:   DefaultSkipTLS != "false",
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("配置文件不存在，使用默认配置")
			return cfg, nil
		}
		return cfg, err
	}
	if err := json5.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	fmt.Printf("已加载配置文件: %s\n", path)
	return cfg, nil
}
