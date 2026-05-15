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

	"github.com/gin-gonic/gin"

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

func proxyHandler(client *http.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		urlString := c.Query("url")
		originURL, err := parseOriginURL(urlString)
		if err != nil {
			c.String(http.StatusBadRequest, "url参数错误")
			return
		}

		req, _ := http.NewRequest(c.Request.Method, urlString, c.Request.Body)
		copyRequestHeaders(c.Request.Header, req.Header)

		resp, err := client.Do(req)
		if err != nil {
			log.Println(err)
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		defer resp.Body.Close()

		status := copyResponseHeaders(resp.Header, c.Writer.Header(), resp.StatusCode)
		modifyLocation(c, originURL.String())
		c.Writer.WriteHeader(status)

		buf := make([]byte, 32*1024)
		_, _ = io.CopyBuffer(c.Writer, resp.Body, buf)
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

func modifyLocation(c *gin.Context, origin string) {
	rawLocation := strings.TrimSpace(c.Writer.Header().Get("Location"))
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
	c.Writer.Header().Del("Location")
	c.Writer.Header().Set("tun-Location", location)
	c.Writer.Header().Set("tun-Location-Proxy", buildProxyURL(location))
}

func buildProxyURL(uri string) string {
	return proxyPath + "?url=" + url.QueryEscape(uri)
}

func isAbsolutePath(uri string) bool { return strings.HasPrefix(uri, "/") }
func isFullURL(uri string) bool {
	return strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://")
}
