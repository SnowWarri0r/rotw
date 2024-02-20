package rotw

// createTime 获取文件创建时间
func createTime(info *FileInfo) int64 {
	stat := info.Sys().(*syscall.Win32FileAttributeData)
	return int64(stat.CreationTime)
}
