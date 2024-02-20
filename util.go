package rotw

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var nowFunc = time.Now
var suffixRegexp = regexp.MustCompile(`\.[\d_-]+`)

// setNowFunc 设置当前时间函数，用于测试
func setNowFunc(f func() time.Time) {
	nowFunc = f
}

// keepDirs 确保目录存在，如果存在的是文件则重命名
func keepDirs(dir string) error {
	info, errStat := os.Stat(dir)
	// 可以查到信息，并且是目录，直接返回
	if errStat == nil && info.IsDir() {
		return nil
	}
	// 可以查到信息，但是不是目录，则重命名
	if errStat == nil {
		rename := dir + ".old" + nowFunc().Format("2006-01-02T15:04:05")
		if err := os.Rename(dir, rename); err != nil {
			// 如果是文件不存在的错误，说明可能已经重命名过了，这种错误可以忽略
			if !os.IsNotExist(err) {
				return err
			}
		}
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		// 如果是已经存在的错误，说明可能已经创建过了，这种错误可以忽略
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

// getExpireFiles 获取过期文件列表
func getExpireFiles(path string, keep int) ([]string, error) {
	pattern := path + ".*"
	matches, errGlob := filepath.Glob(pattern)
	if errGlob != nil {
		return nil, errGlob
	}
	// 文件少于等于keep个，没有过期文件
	if len(matches) <= keep {
		return nil, nil
	}
	fileInfos := make([]os.FileInfo, 0, len(matches))
	prefix := filepath.Base(path)
	for i := 0; i < len(matches); i++ {
		name := matches[i]
		info, err := os.Stat(name)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			_, _ = fmt.Fprintf(os.Stderr, "stat file fail, err=%v", err)
			continue
		}
		// 不是文件，跳过
		if info.IsDir() {
			continue
		}
		// 文件名规则不正确，跳过
		if !isFilenameMatch(prefix, info.Name()) {
			continue
		}
		fileInfos = append(fileInfos, info)
	}
	// 按文件创建时间排序
	sort.Slice(fileInfos, func(i int, j int) bool {
		return createTime(fileInfos[i]) < createTime(fileInfos[j])
	})

	ret := make([]string, 0)
	dir := filepath.Dir(path)
	for i := 0; i < len(fileInfos)-keep; i++ {
		name := filepath.Join(dir, fileInfos[i].Name())
		ret = append(ret, name)
	}
	return ret, nil
}

// isFilenameMatch 检查文件名是否满足前缀，且后缀格式为 \.[\d_-]+ 的正则表达式
func isFilenameMatch(prefix string, name string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	suffix := strings.TrimPrefix(name, prefix)
	if len(suffix) == 0 || suffix[0] != '.' {
		return false
	}
	return suffixRegexp.MatchString(name)
}
