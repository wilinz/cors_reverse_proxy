package server

import "strings"

func validBearer(authorizationHeader, authKey string) bool {
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(strings.ToLower(authorizationHeader), strings.ToLower(bearerPrefix)) {
		return false
	}
	token := strings.TrimSpace(authorizationHeader[len(bearerPrefix):])
	return token == authKey
}
