package utils

import (
	"path/filepath"
	"runtime"
)

// GetProjectRoot 获取项目根目录的绝对路径
func GetProjectRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		// 向上两级（internal/utils）得到项目根目录
		dir := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
		return dir
	}
	return ""
}
