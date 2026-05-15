package server

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cors_reverse_proxy/internal/config"
)

const proxyPath = "/proxy"

func newHTTPClient(cfg config.Config) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.SkipTLS},
		Proxy:           http.ProxyFromEnvironment,
	}
	if proxy := strings.TrimSpace(cfg.HttpProxy); proxy != "" {
		if proxyURL, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		} else {
			log.Printf("http_proxy 格式错误，将使用默认代理: %v\n", err)
		}
	}
	return &http.Client{
		Transport: transport,
		Timeout:   5 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func proxyHandler(client *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		urlString := r.URL.Query().Get("url")
		originURL, err := parseOriginURL(urlString)
		if err != nil {
			http.Error(w, "url参数错误", http.StatusBadRequest)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), r.Method, urlString, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		copyRequestHeaders(r.Header, req.Header)

		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		status := copyResponseHeaders(resp.Header, w.Header(), resp.StatusCode)
		modifyLocation(w.Header(), originURL.String())
		w.WriteHeader(status)

		buf := make([]byte, 32*1024)
		_, _ = io.CopyBuffer(w, resp.Body, buf)
	}
}

func parseOriginURL(urlString string) (*url.URL, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	u.Path = "/"
	u.RawQuery = ""
	return u, nil
}

func modifyLocation(h http.Header, origin string) {
	rawLocation := strings.TrimSpace(h.Get("Location"))
	if rawLocation == "" {
		return
	}
	originURL, _ := url.Parse(origin)
	location := rawLocation
	switch {
	case strings.HasPrefix(location, "//"):
		if originURL != nil && originURL.Scheme != "" {
			location = fmt.Sprintf("%s:%s", originURL.Scheme, location)
		}
	case isFullURL(location):
	case isAbsolutePath(location):
		location = origin + location
	default:
		location = origin + "/" + strings.TrimPrefix(location, "/")
	}
	h.Del("Location")
	h.Set("tun-Location", location)
	h.Set("tun-Location-Proxy", buildProxyURL(location))
}

func buildProxyURL(uri string) string {
	return proxyPath + "?url=" + url.QueryEscape(uri)
}

func isAbsolutePath(uri string) bool { return strings.HasPrefix(uri, "/") }
func isFullURL(uri string) bool {
	return strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://")
}
