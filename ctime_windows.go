package rotw

import (
	"os"
	"syscall"
)

// createTime 获取文件创建时间
func createTime(info os.FileInfo) int64 {
	stat := info.Sys().(*syscall.Win32FileAttributeData)
	return stat.CreationTime.Nanoseconds() / 1e6
}
