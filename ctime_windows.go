package rotw

import (
	"os"
	"syscall"
)

// createTime returns the creation time of a file.
func createTime(info os.FileInfo) int64 {
	stat := info.Sys().(*syscall.Win32FileAttributeData)
	return stat.CreationTime.Nanoseconds() / 1e6
}
