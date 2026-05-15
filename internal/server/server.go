package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"cors_reverse_proxy/internal/config"
)

// Run starts the HTTP server. Blocks until error.
func Run(cfg config.Config) error {
	client := newHTTPClient(cfg)

	r := gin.Default()
	r.Use(corsAndAuth(cfg.Token))

	r.GET("/lanip", lanIPHandler)
	r.GET("/kill", killHandler)
	r.Any(proxyPath, proxyHandler(client))

	fmt.Printf("运行在 http://%s\n", cfg.Listening)
	return r.Run(cfg.Listening)
}

func corsAndAuth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", origin)
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")

		requestHeaders := c.Request.Header.Get("Access-Control-Request-Headers")
		if requestHeaders == "" {
			requestHeaders = "Content-Type, Authorization"
		}
		h.Set("Access-Control-Allow-Headers", requestHeaders)
		h.Set("Access-Control-Max-Age", "86400")
		h.Set("Access-Control-Allow-Credentials", "true")
		h.Set("Access-Control-Expose-Headers", "tun-Location, tun-Location-Proxy, tun-set-cookie, tun-status")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		h.Set("Cache-Control", "no-store, no-cache, must-revalidate")
		h.Set("Pragma", "no-cache")
		h.Set("Expires", "0")

		if !validBearer(c.GetHeader("Authorization"), token) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "未认证，请更新App: bearer 认证失败",
			})
			return
		}
		c.Next()
	}
}
