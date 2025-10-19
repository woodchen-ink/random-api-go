package utils

import (
	"strings"
)

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
