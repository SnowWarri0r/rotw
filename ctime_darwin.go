package rotw

// createTime returns the creation time of a file.
func createTime(info os.FileInfo) int64 {
	stat := info.Sys().(*syscall.Stat_t)
	return stat.Birthtimespec.Sec
}
