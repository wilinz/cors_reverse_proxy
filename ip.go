package main // getLocalIP 获取本机 IP 地址（跨平台）
import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net"
	"strings"
)

func GetLanIPHandler(c *gin.Context) {
	ip, err := getLocalIP()
	if err != nil {
		c.JSON(500, gin.H{
			"code": -1,
			"msg":  err.Error(),
			"ip":   "",
		})
		return
	}

	c.JSON(200, gin.H{
		"code": 0,
		"msg":  "success",
		"ip":   ip,
	})
}

// 优先级: 以太网 > WiFi > 其他任意 IPv4（排除127.0.0.x）
func getLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var ethernetIP, wifiIP, fallbackIP string

	for _, iface := range interfaces {
		// 跳过未启用或回环接口
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP.To4()
			// 只要 IPv4，排除回环
			if ip == nil || ip.IsLoopback() {
				continue
			}

			ipStr := ip.String()

			// 排除 127.0.0.x
			if strings.HasPrefix(ipStr, "127.") {
				continue
			}

			ifaceName := strings.ToLower(iface.Name)

			// 判断是以太网
			// Windows: "以太网", "Ethernet", "本地连接"
			// Linux: eth0, enp3s0
			// macOS: en0
			isEthernet := strings.Contains(ifaceName, "eth") ||
				strings.Contains(ifaceName, "en") ||
				strings.Contains(ifaceName, "ethernet") ||
				strings.Contains(ifaceName, "以太网") ||
				strings.Contains(ifaceName, "本地连接") ||
				(strings.HasPrefix(ifaceName, "en") && len(ifaceName) <= 4)

			// 判断是 WiFi
			// Windows: "Wi-Fi", "WLAN", "无线网络连接"
			// Linux: wlan0
			isWiFi := strings.Contains(ifaceName, "wlan") ||
				strings.Contains(ifaceName, "wifi") ||
				strings.Contains(ifaceName, "wi-fi") ||
				strings.Contains(ifaceName, "wl") ||
				strings.Contains(ifaceName, "无线") ||
				strings.Contains(ifaceName, "wireless")

			if isEthernet && ethernetIP == "" {
				ethernetIP = ipStr
			} else if isWiFi && wifiIP == "" {
				wifiIP = ipStr
			} else if fallbackIP == "" {
				fallbackIP = ipStr
			}
		}
	}

	// 返回优先级最高的
	if ethernetIP != "" {
		return ethernetIP, nil
	}
	if wifiIP != "" {
		return wifiIP, nil
	}
	if fallbackIP != "" {
		return fallbackIP, nil
	}

	return "", fmt.Errorf("未找到可用的 IPv4 地址")
}
