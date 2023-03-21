package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/wilinz/go-filex"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var proxyPath = "/proxy"

var (
	apiKey = ""
)

func main() {

	appDir := filex.NewFile(os.Args[0]).ParentFile()
	apikeyFile := filex.NewFile2(appDir, "openaikey.txt")
	if !apikeyFile.IsExist() {
		log.Panic("请在把apikey放在程序目录下的openaikey.txt")
	}
	var err error
	apiKey, err = apikeyFile.ReadAllString()
	apiKey = strings.Trim(apiKey, "\r\n ")
	if err == nil {
		log.Println("apikey配置成功")
	}

	httpClient := &http.Client{
		//Transport: tr,
		Timeout: time.Minute * 5, //超时时间
	}

	r := gin.Default()

	//"http://127.0.0.1:9999/proxy?url=https://www.baidu.com"
	r.Any(proxyPath, func(c *gin.Context) {

		c.Writer.Header().Set("Access-Control-Allow-Origin", c.Request.Header.Get("Origin"))
		c.Writer.Header().Set("Access-Control-Allow-Methods", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "X-Referer, X-User-Agent")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == http.MethodOptions {
			c.Writer.WriteHeader(200)
			return
		}
		urlString := c.Query("url")

		uri, _ := url.Parse(urlString)

		originUrl, err := parseOriginUrl(urlString)

		if err != nil {
			c.String(http.StatusBadRequest, "url参数错误")
			return
		}

		req, _ := http.NewRequest(c.Request.Method, urlString, c.Request.Body)
		copyRequestHeader(c, req)
		if req.Header.Get("Authorization") == "" && uri.Host == "api.openai.com" {
			req.Header.Set("Authorization", fmt.Sprint("Bearer ", apiKey))
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			log.Println(err)
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		copyResponseHeader(c, resp)
		modifyLocation(c, originUrl.String())
		io.Copy(c.Writer, resp.Body)
	})
	fmt.Println("运行在 9999 端口")
	err1 := r.Run(":9999")
	if err1 != nil {
		log.Fatalln(err1)
		return
	}
}

func parseOriginUrl(urlString string) (*url.URL, error) {
	originUrl, err := url.Parse(urlString)
	originUrl.Path = "/"
	originUrl.RawQuery = ""
	return originUrl, err
}

func modifyLocation(c *gin.Context, origin string) {
	location := c.Writer.Header().Get("Location")
	if isAbsolutePath(location) {
		location = buildProxyUrl(origin + location)
	} else if isFullURL(location) {
		location = buildProxyUrl(location)
	}
	c.Writer.Header().Set("Location", location)
}

func copyResponseHeader(c *gin.Context, resp *http.Response) {
	cookies := resp.Cookies()
	for _, cookie := range cookies {
		cookie.SameSite = http.SameSiteNoneMode
		cookie.Secure = true
		c.Writer.Header().Add("Set-Cookie", cookie.String())
	}
	c.Writer.WriteHeader(resp.StatusCode)
	for k, vList := range resp.Header {
		if k != "Set-Cookie" {
			for _, v := range vList {
				c.Writer.Header().Add(k, v)
			}
		}
	}
}

func copyRequestHeader(c *gin.Context, req *http.Request) {
	for k, vList := range c.Request.Header {
		newKey := k
		if strings.HasPrefix(k, "X-") {
			newKey = k[len("X-"):]
			c.Request.Header.Del(newKey)
			req.Header.Del(newKey)
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
