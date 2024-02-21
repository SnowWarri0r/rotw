package rotw

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Option func(*RotateWriterConfig)

// RotateWriterConfig 文件分割写入器配置
type RotateWriterConfig struct {
	// 最多保留多少个文件, Optional, 默认0，即不删除文件
	KeepFiles int
	// 默认文件分割规则
	//
	// 1min, 5min, 10min, 30min, hour, day
	//
	// 可以使用 AddRotateRule 添加自定义规则, Optional, 默认规则为 no
	Rule string
	// 文件路径: eg: xxx/xxx.log
	LogPath string
	// 检查文件是否打开的间隔时间, Optional, 默认1s
	CheckSpan time.Duration
}

func (rw *RotateWriterConfig) check() error {
	if len(rw.Rule) == 0 {
		rw.Rule = "no"
	}
	if len(rw.LogPath) == 0 {
		return errors.New("log path is empty")
	}
	if rw.CheckSpan <= 0 {
		rw.CheckSpan = time.Second * 1
	}
	return nil
}

// RotateWriter 文件分割写入器
type RotateWriter interface {
	io.WriteCloser
}

type rotateWriter struct {
	cfg *RotateWriterConfig
	// 当前文件信息
	fileInfo os.FileInfo
	// 文件分割信息生成器
	rig RotateInfoGenerator
	// 当前文件
	file *os.File
	mux  sync.Mutex
	// 关闭信号，用于通知检查文件是否打开的协程退出
	closed chan struct{}
}

// NewRotateWriterWithOpt 创建文件分割写入器
func NewRotateWriterWithOpt(logPath string, opts ...Option) (RotateWriter, error) {
	cfg := &RotateWriterConfig{
		LogPath: logPath,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	rw, err := NewRotateWriter(cfg)
	if err != nil {
		return nil, err
	}
	return rw, nil
}

// NewRotateWriter 创建文件分割写入器
func NewRotateWriter(cfg *RotateWriterConfig) (RotateWriter, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	if err := cfg.check(); err != nil {
		return nil, err
	}
	// 创建文件信息生成器
	rig, errRig := NewRotateInfoGenerator(cfg.Rule, cfg.LogPath)
	if errRig != nil {
		return nil, errRig
	}
	rw := &rotateWriter{
		cfg:    cfg,
		rig:    rig,
		closed: make(chan struct{}),
	}
	if err := rw.init(); err != nil {
		errClose := rw.Close()
		_, _ = fmt.Fprintf(os.Stderr, "close rotate writer error, err=%v\n", errClose)
		return nil, err
	}
	return rw, nil
}

func (r *rotateWriter) init() error {
	cfg := r.cfg
	rig := r.rig
	// 检查文件是否打开
	if err := r.check(rig.Get()); err != nil {
		return err
	}
	// 添加回调，当文件信息变化时，检查文件是否打开
	rig.AddCallback(func(val rotateInfo) {
		err := r.check(val)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "check file error, err=%v\n", err)
		}
	})
	// KeepFiles > 0 时，开启清理过期文件协程
	if cfg.KeepFiles > 0 {
		rig.AddCallbackWithCtx(func(ctx context.Context, val rotateInfo) {
			r.clean(ctx)
		})
		// 启动时清理过期文件
		go r.clean(context.Background())
	}
	// CheckSpan > 0 时，开启检查文件是否打开的协程
	if cfg.CheckSpan > 0 {
		go r.doCheck(cfg.CheckSpan, rig)
	}
	return nil
}

// Write 写入数据
func (r *rotateWriter) Write(p []byte) (n int, err error) {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.file.Write(p)
}

// Close 关闭文件分割写入器
func (r *rotateWriter) Close() error {
	close(r.closed)
	r.rig.Stop()
	if r.file == nil {
		return nil
	}
	return r.file.Close()
}

// clean 清理过期文件
func (r *rotateWriter) clean(ctx context.Context) {
	info := r.rig.Get()
	files, err := getExpireFiles(info.RawPath, r.cfg.KeepFiles)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "get expire files error, err=%v\n", err)
		return
	}
	if len(files) == 0 {
		return
	}
	// 每秒删除一个文件，减少删除文件时的压力
	tm := time.NewTimer(time.Second)
	defer tm.Stop()
	for i := 0; i < len(files); i++ {
		select {
		case <-ctx.Done():
			return
		case <-tm.C:
		}
		now := nowFunc()
		name := files[i]
		if errRemove := os.Remove(name); errRemove != nil {
			_, _ = fmt.Fprintf(os.Stderr, "remove file %s error, err=%v\n", name, err)
		}
		cost := time.Since(now)
		fmt.Printf("[%v] >>> remove file %s, cost: %v\n", nowFunc().Format(time.StampMicro), name, cost)
		tm.Reset(time.Second)
	}
}

func (r *rotateWriter) doCheck(span time.Duration, rig RotateInfoGenerator) {
	ticker := time.NewTicker(span)
	defer ticker.Stop()
	for {
		select {
		case <-r.closed:
			return
		case <-ticker.C:
			if err := r.check(rig.Get()); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "check file error, err=%v\n", err)
			}
		}
	}
}

// check 检查文件是否存在，不存在则创建目录和文件, 文件存在则检查文件是否被修改过
func (r *rotateWriter) check(info rotateInfo) error {
	r.mux.Lock()
	fileExists := r.isFileExists(info.RotatePath)
	r.mux.Unlock()

	if !fileExists {
		// 文件不存在，则创建目录
		dir := filepath.Dir(info.RotatePath)
		if err := keepDirs(dir); err != nil {
			return err
		}
	}

	r.mux.Lock()
	defer r.mux.Unlock()
	// 文件存在且没有修改过，则直接返回
	if r.file != nil && fileExists {
		return nil
	}
	// 上一个文件描述符存在，则关闭
	if r.file != nil {
		errClose := r.file.Close()
		if errClose != nil {
			_, _ = fmt.Fprintf(os.Stderr, "close file %s error, err=%v\n", r.file.Name(), errClose)
		}
	}
	// 创建新文件/打开文件
	file, err := os.OpenFile(info.RotatePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	// 获取文件信息
	fileStat, err := file.Stat()
	if err != nil {
		return err
	}
	// 更新文件信息
	r.fileInfo = fileStat
	// 更新文件描述符
	r.file = file
	return nil
}

// isFileExists 判断文件是否已经存在，且与当前文件信息一致
func (r *rotateWriter) isFileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
		_, _ = fmt.Fprintf(os.Stderr, "stat %s error, err=%v\n", filename, err)
		return false
	}
	return os.SameFile(info, r.fileInfo)
}

func WithKeepFiles(num int) Option {
	return func(rw *RotateWriterConfig) {
		rw.KeepFiles = num
	}
}

func WithRule(rule string) Option {
	return func(rw *RotateWriterConfig) {
		rw.Rule = rule
	}
}

func WithCheckSpan(span time.Duration) Option {
	return func(rw *RotateWriterConfig) {
		rw.CheckSpan = span
	}
}
