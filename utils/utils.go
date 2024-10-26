package utils

import (
	"net"
	"net/http"
	"strings"
)

func GetRealIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	ip = r.Header.Get("X-Forwarded-For")
	if ip != "" {
		ips := strings.Split(ip, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// 添加一个处理URL路径的工具函数
func JoinURLPath(parts ...string) string {
	// 过滤空字符串
	var nonEmptyParts []string
	for _, part := range parts {
		if part != "" {
			// 去除首尾的"/"
			part = strings.Trim(part, "/")
			if part != "" {
				nonEmptyParts = append(nonEmptyParts, part)
			}
		}
	}

	// 用"/"连接所有部分
	return strings.Join(nonEmptyParts, "/")
}
