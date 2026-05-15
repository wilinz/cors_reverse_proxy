package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"cors_reverse_proxy/internal/config"
)

// Run starts the HTTP server. Blocks until error.
func Run(cfg config.Config) error {
	client := newHTTPClient(cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/lanip", lanIPHandler)
	mux.HandleFunc("/kill", killHandler)
	mux.HandleFunc(proxyPath, proxyHandler(client))

	handler := logging(corsAndAuth(cfg.Token, mux))

	fmt.Printf("运行在 http://%s\n", cfg.Listening)
	return http.ListenAndServe(cfg.Listening, handler)
}

func corsAndAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", origin)
		h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")

		requestHeaders := r.Header.Get("Access-Control-Request-Headers")
		if requestHeaders == "" {
			requestHeaders = "Content-Type, Authorization"
		}
		h.Set("Access-Control-Allow-Headers", requestHeaders)
		h.Set("Access-Control-Max-Age", "86400")
		h.Set("Access-Control-Allow-Credentials", "true")
		h.Set("Access-Control-Expose-Headers", "tun-Location, tun-Location-Proxy, tun-set-cookie, tun-status")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		h.Set("Cache-Control", "no-store, no-cache, must-revalidate")
		h.Set("Pragma", "no-cache")
		h.Set("Expires", "0")

		if !validBearer(r.Header.Get("Authorization"), token) {
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"error": "未认证，请更新App: bearer 认证失败",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.RequestURI())
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
