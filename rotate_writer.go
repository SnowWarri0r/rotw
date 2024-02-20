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

// RotateWriterOption 文件分割写入器配置
type RotateWriterOption struct {
	// 最多保留多少个文件
	KeepFiles int
	// 分割文件名生成器
	Rg RotateGenerator
	// 检查文件是否打开的间隔时间
	CheckSpan time.Duration
}

func (rw *RotateWriterOption) check() error {
	if rw.Rg == nil {
		return errors.New("rotate generator is nil")
	}
	return nil
}

// RotateWriter 文件分割写入器
type RotateWriter interface {
	io.WriteCloser
}

type rotateWriter struct {
	opt *RotateWriterOption
	// 当前文件信息
	fileInfo os.FileInfo
	// 当前文件
	file *os.File
	mux  sync.Mutex
	// 关闭信号，用于通知检查文件是否打开的协程退出
	closed chan struct{}
}

// NewRotateWriter 创建文件分割写入器
func NewRotateWriter(opt *RotateWriterOption) (RotateWriter, error) {
	if opt == nil {
		return nil, errors.New("option is nil")
	}
	if err := opt.check(); err != nil {
		return nil, err
	}
	rw := &rotateWriter{
		opt:    opt,
		closed: make(chan struct{}),
	}
	if err := rw.init(); err != nil {
		errClose := rw.Close()
		_, _ = fmt.Fprintf(os.Stderr, "close rotate writer error: %v\n", errClose)
		return nil, err
	}
	return rw, nil
}

func (r *rotateWriter) init() error {
	opt := r.opt
	rg := opt.Rg
	// 检查文件是否打开
	if err := r.check(rg.Get()); err != nil {
		return err
	}
	// 添加回调，当文件信息变化时，检查文件是否打开
	rg.AddCallback(func(val rotateInfo) {
		err := r.check(val)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "check file error: %v\n", err)
		}
	})
	// KeepFiles > 0 时，开启清理过期文件协程
	if opt.KeepFiles > 0 {
		rg.AddCallbackWithCtx(func(ctx context.Context, val rotateInfo) {
			r.clean(ctx)
		})
		// 启动时清理过期文件
		go r.clean(context.Background())
	}
	// CheckSpan > 0 时，开启检查文件是否打开的协程
	if opt.CheckSpan > 0 {
		go r.doCheck(opt.CheckSpan, rg)
	}
	return nil
}

// Write 写入数据
func (r *rotateWriter) Write(p []byte) (n int, err error) {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.file.Write(p)
}

// Close 关闭文件
func (r *rotateWriter) Close() error {
	close(r.closed)
	r.opt.Rg.Stop()
	if r.file == nil {
		return nil
	}
	return r.file.Close()
}

// clean 清理过期文件
func (r *rotateWriter) clean(ctx context.Context) {
	info := r.opt.Rg.Get()
	files, err := getExpireFiles(info.RawPath, r.opt.KeepFiles)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "get expire files error: %v\n", err)
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
			_, _ = fmt.Fprintf(os.Stderr, "remove file %s error: %v\n", name, err)
		}
		cost := time.Since(now)
		fmt.Printf("[%v] >>> remove file %s, cost: %v\n", nowFunc().Format(time.StampMicro), name, cost)
		tm.Reset(time.Second)
	}
}

func (r *rotateWriter) doCheck(span time.Duration, rg RotateGenerator) {
	ticker := time.NewTicker(span)
	defer ticker.Stop()
	for {
		select {
		case <-r.closed:
			return
		case <-ticker.C:
			if err := r.check(rg.Get()); err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "check file error: %v\n", err)
			}
		}
	}
}

// check 检查文件是否存在，不存在则创建文件和目录, 文件存在则检查文件是否被修改过
func (r *rotateWriter) check(info rotateInfo) error {
	r.mux.Lock()
	fileExists := r.isFileExists(info.RotatePath)
	r.mux.Unlock()

	if !fileExists {
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
		_ = r.file.Close()
	}
	// 创建新文件
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
		_, _ = fmt.Fprintf(os.Stderr, "stat %s error: %v\n", filename, err)
		return false
	}
	return os.SameFile(info, r.fileInfo)
}
