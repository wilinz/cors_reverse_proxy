package main

import (
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
	Cert      string `json:"cert"`
	Key       string `json:"key"`
	Port      int    `json:"port"`
	OpenaiKey string `json:"openai_key"`
	AuthKey   string `json:"auth_key"`
}

func main() {

	appDir := filex.NewFile(os.Args[0]).ParentFile()
	configFile := filex.NewFile2(appDir, "config.json")
	if !configFile.IsExist() {
		temp := filex.NewFile2(appDir, "config.temp.json")
		f, _ := temp.Create()
		byteArr, _ := json.MarshalIndent(Config{
			Tls:       false,
			Cert:      "",
			Key:       "",
			Port:      10010,
			OpenaiKey: "",
			AuthKey:   "",
		}, "", "    ")
		f.Write(byteArr)
		f.Close()
		log.Panic("请配置放在程序目录下的 config.json")
	}
	f, _ := configFile.Open()
	b, _ := ioutil.ReadAll(f)
	json.Unmarshal(b, &config)

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

		authKey := c.Query("key")
		if authKey != config.AuthKey {
			c.String(http.StatusUnauthorized, "未认证，请更新App")
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
		if uri.Host == "api.openai.com" {
			req.Header.Set("Authorization", fmt.Sprint("Bearer ", config.OpenaiKey))
		}

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

	addr := fmt.Sprintf(":%d", config.Port)
	fmt.Printf("运行在 %d 端口\n", config.Port)
	if config.Tls {
		err := r.RunTLS(addr, config.Cert, config.Key)
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
