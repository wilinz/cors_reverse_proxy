package server

import (
	"net/http"
	"strconv"
	"strings"
)

const tunPrefix = "tun-"

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

func shouldForwardByDefault(header string) bool {
	_, ok := defaultForwardHeaders[header]
	return ok
}

func isCorsHeader(header string) bool {
	return strings.HasPrefix(strings.ToLower(header), "access-control-")
}

func copyRequestHeaders(src, dst http.Header) {
	tunHeaders := make(map[string]struct{})
	for k := range src {
		if len(k) > len(tunPrefix) && strings.EqualFold(k[:len(tunPrefix)], tunPrefix) {
			tunHeaders[strings.ToLower(k[len(tunPrefix):])] = struct{}{}
		}
	}
	for k, vList := range src {
		lowered := strings.ToLower(k)
		isTunHeader := len(k) > len(tunPrefix) && strings.EqualFold(k[:len(tunPrefix)], tunPrefix)
		if !isTunHeader && !shouldForwardByDefault(lowered) {
			continue
		}
		newKey := k
		if isTunHeader {
			newKey = k[len(tunPrefix):]
		} else if _, exists := tunHeaders[lowered]; exists {
			continue
		}
		for _, v := range vList {
			dst.Add(newKey, v)
		}
	}
}

func copyResponseHeaders(src, dst http.Header, statusCode int) int {
	isRedirect := statusCode >= 300 && statusCode < 400
	finalStatus := statusCode
	if isRedirect {
		finalStatus = http.StatusOK
		dst.Set("tun-status", strconv.Itoa(statusCode))
	}
	for k, vList := range src {
		if isCorsHeader(k) {
			continue
		}
		headerKey := k
		if strings.EqualFold(k, "set-cookie") {
			headerKey = "tun-set-cookie"
		}
		for _, v := range vList {
			dst.Add(headerKey, v)
		}
	}
	return finalStatus
}
