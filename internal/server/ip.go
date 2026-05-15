package server

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

func lanIPHandler(w http.ResponseWriter, r *http.Request) {
	ip, err := getLocalIP()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"code": -1, "msg": err.Error(), "ip": "",
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"code": 0, "msg": "success", "ip": ip,
	})
}

// 优先级: 以太网 > WiFi > 其他任意 IPv4（排除 127.0.0.x）
func getLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	var ethernetIP, wifiIP, fallbackIP string

	for _, iface := range interfaces {
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
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ipStr := ip.String()
			if strings.HasPrefix(ipStr, "127.") {
				continue
			}

			name := strings.ToLower(iface.Name)
			isEthernet := strings.Contains(name, "eth") ||
				strings.Contains(name, "en") ||
				strings.Contains(name, "ethernet") ||
				strings.Contains(name, "以太网") ||
				strings.Contains(name, "本地连接") ||
				(strings.HasPrefix(name, "en") && len(name) <= 4)
			isWiFi := strings.Contains(name, "wlan") ||
				strings.Contains(name, "wifi") ||
				strings.Contains(name, "wi-fi") ||
				strings.Contains(name, "wl") ||
				strings.Contains(name, "无线") ||
				strings.Contains(name, "wireless")

			if isEthernet && ethernetIP == "" {
				ethernetIP = ipStr
			} else if isWiFi && wifiIP == "" {
				wifiIP = ipStr
			} else if fallbackIP == "" {
				fallbackIP = ipStr
			}
		}
	}

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
