package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/wilinz/go-filex"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var proxyPath = "/proxy"

var (
	config Config
)

type Config struct {
	Tls       bool   `json:"tls"`
	TLSCert   string `json:"tls_cert"`
	TLSKey    string `json:"tls_key"`
	Listening string `json:"listening"`
	Token     string `json:"token"`
	HttpProxy string `json:"http_proxy"`
	SkipTLS   bool   `json:"skip_tls"`
}

func main() {

	appDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}
	fmt.Println("Current working directory:", appDir)

	configFile := filex.NewFile1(appDir, "config.json5")
	if !configFile.IsExist() {
		temp := filex.NewFile1(appDir, "config.temp.json5")
		f, _ := temp.Create()
		byteArr, _ := json.MarshalIndent(Config{
			Tls:       false,
			TLSCert:   "",
			TLSKey:    "",
			Listening: "0.0.0.0:10010",
			Token:     "",
			HttpProxy: "",
			SkipTLS:   false,
		}, "", "    ")
		f.Write(byteArr)
		print(f.Name())
		f.Close()
		log.Panic("请配置放在程序目录下的 config.json5")
	}
	f, _ := configFile.Open()
	b, _ := ioutil.ReadAll(f)
	json.Unmarshal(b, &config)

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipTLS},
		Proxy:           http.ProxyFromEnvironment,
	}
	if proxy := strings.TrimSpace(config.HttpProxy); proxy != "" {
		if proxyURL, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		} else {
			log.Printf("http_proxy 格式错误，将使用默认代理: %v\n", err)
		}
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   time.Minute * 5, //超时时间
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Do not follow redirects automatically; return the last response to the client.
			return http.ErrUseLastResponse
		},
	}

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "http://localhost:3000" // 设置默认或拒绝
		}

		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")

		requestHeaders := c.Request.Header.Get("Access-Control-Request-Headers")
		if requestHeaders == "" {
			requestHeaders = "Content-Type, Authorization"
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", requestHeaders)
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "tun-Location, tun-Location-Proxy, tun-set-cookie, tun-status")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Writer.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
		c.Writer.Header().Set("Pragma", "no-cache")
		c.Writer.Header().Set("Expires", "0")

		authHeader := c.GetHeader("Authorization") // 注意大小写
		if !validBearer(authHeader, config.Token) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "未认证，请更新App: bearer 认证失败",
			})
			return
		}

		c.Next()
	})

	// 获取本机 IP
	r.GET("/lanip", GetLanIPHandler)

	//"http://127.0.0.1:9999/proxy?url=https://www.baidu.com"
	r.Any(proxyPath, func(c *gin.Context) {

		urlString := c.Query("url")

		originUrl, err := parseOriginUrl(urlString)

		if err != nil {
			c.String(http.StatusBadRequest, "url参数错误")
			return
		}

		req, _ := http.NewRequest(c.Request.Method, urlString, c.Request.Body)
		copyRequestHeader(c, req)

		resp, err := httpClient.Do(req)
		if err != nil {
			log.Println(err)
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		copyResponseHeader(c, resp)
		modifyLocation(c, originUrl.String())
		buf := make([]byte, 128)
		io.CopyBuffer(c.Writer, resp.Body, buf)
	})

	addr := fmt.Sprintf("%s", config.Listening)
	fmt.Printf("运行在 %s \n", config.Listening)
	if config.Tls {
		err := r.RunTLS(addr, config.TLSCert, config.TLSKey)
		if err != nil {
			log.Fatalln(err)
			return
		}
	} else {
		err1 := r.Run(addr)
		if err1 != nil {
			log.Fatalln(err1)
			return
		}
	}

}

func parseOriginUrl(urlString string) (*url.URL, error) {
	originUrl, err := url.Parse(urlString)
	originUrl.Path = "/"
	originUrl.RawQuery = ""
	return originUrl, err
}

func modifyLocation(c *gin.Context, origin string) {
	rawLocation := strings.TrimSpace(c.Writer.Header().Get("Location"))
	if rawLocation == "" {
		return
	}

	originURL, _ := url.Parse(origin)
	location := rawLocation

	switch {
	case strings.HasPrefix(location, "//"): // protocol-relative URL
		if originURL != nil && originURL.Scheme != "" {
			location = fmt.Sprintf("%s:%s", originURL.Scheme, location)
		}
	case isFullURL(location):
	case isAbsolutePath(location):
		location = origin + location
	default: // relative path like "foo/bar"
		location = origin + "/" + strings.TrimPrefix(location, "/")
	}

	locationProxy := buildProxyUrl(location)
	// 将上游 Location 重命名为 tun-Location 返回给前端
	c.Writer.Header().Del("Location")
	c.Writer.Header().Set("tun-Location", location)
	c.Writer.Header().Set("tun-Location-Proxy", locationProxy)
}

func copyResponseHeader(c *gin.Context, resp *http.Response) {
	isRedirect := resp.StatusCode >= 300 && resp.StatusCode < 400
	statusCode := resp.StatusCode
	if isRedirect {
		statusCode = http.StatusOK
		c.Writer.Header().Set("tun-status", fmt.Sprintf("%d", resp.StatusCode))
	}
	for k, vList := range resp.Header {
		if isCorsHeader(k) {
			// Drop upstream CORS headers to avoid clashing with our own values
			continue
		}
		headerKey := k
		if strings.EqualFold(k, "set-cookie") {
			headerKey = "tun-set-cookie"
		}
		for _, v := range vList {
			c.Writer.Header().Add(headerKey, v)
		}
	}
	c.Writer.WriteHeader(statusCode)
}

func copyRequestHeader(c *gin.Context, req *http.Request) {
	tunHeaders := make(map[string]struct{})
	for k := range c.Request.Header {
		if len(k) > len("tun-") && strings.EqualFold(k[:len("tun-")], "tun-") {
			tunHeaders[strings.ToLower(k[len("tun-"):])] = struct{}{}
		}
	}
	for k, vList := range c.Request.Header {
		lowered := strings.ToLower(k)
		// Forward whitelisted headers by default, or headers explicitly prefixed with tun-
		isTunHeader := len(k) > len("tun-") && strings.EqualFold(k[:len("tun-")], "tun-")
		if !isTunHeader && !shouldForwardByDefault(lowered) {
			continue
		}
		newKey := k
		if isTunHeader {
			newKey = k[len("tun-"):]
		} else if _, exists := tunHeaders[lowered]; exists {
			// Skip default forwarding when an explicit tun- version is provided to avoid duplicates
			continue
		}
		for _, v := range vList {
			req.Header.Add(newKey, v)
		}
	}
}

func buildProxyUrl(uri string) string {
	return proxyPath + "?url=" + url.QueryEscape(uri)
}
func isAbsolutePath(uri string) bool {
	return strings.HasPrefix(uri, "/")
}

func isFullURL(uri string) bool {
	return strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://")
}

func validBearer(authorizationHeader, authKey string) bool {
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(strings.ToLower(authorizationHeader), strings.ToLower(bearerPrefix)) {
		return false
	}
	token := strings.TrimSpace(authorizationHeader[len(bearerPrefix):])
	return token == authKey
}

func isCorsHeader(header string) bool {
	return strings.HasPrefix(strings.ToLower(header), "access-control-")
}

func shouldForwardByDefault(header string) bool {
	_, ok := defaultForwardHeaders[header]
	return ok
}

var defaultForwardHeaders = map[string]struct{}{
	"content-type":    {},
	"content-length":  {},
	"referer":         {},
	"user-agent":      {},
	"accept":          {},
	"cookie":          {},
	"accept-encoding": {},
	"keep-alive":      {},
}
