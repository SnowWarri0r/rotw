package rotw

import (
	"context"
	"errors"
	"fmt"
	"time"
)

type rotateInfo struct {
	// 原始文件路径
	RawPath string
	// 分割后的文件路径
	RotatePath string
}

// RotateInfoGenerator 文件分割信息生成器
type RotateInfoGenerator interface {
	// Get 获取当前文件路径
	Get() rotateInfo
	// AddCallback 添加回调函数
	AddCallback(func(rotateInfo))
	// AddCallbackWithCtx 添加回调函数，并传入生成器的上下文，用于用户控制中断任务
	AddCallbackWithCtx(func(context.Context, rotateInfo))
	// Stop 停止生成器，此操作会cancel生成器的上下文，并停止生成器
	Stop()
}

type rotateInfoGenerator struct {
	g Generator
}

// ErrInvalidRule 无效规则错误
var ErrInvalidRule = errors.New("invalid rule")

// NewRotateInfoGenerator 创建文件分割信息生成器
func NewRotateInfoGenerator(rule string, filePath string) (RotateInfoGenerator, error) {
	if r, ok := defaultRotateRule[rule]; ok {
		fn := func() any {
			return rotateInfo{
				RawPath:    filePath,
				RotatePath: filePath + r.SuffixFunc(),
			}
		}
		return &rotateInfoGenerator{
			g: NewGenerator(r.Span, fn),
		}, nil
	}
	return nil, ErrInvalidRule
}

func (r *rotateInfoGenerator) Get() rotateInfo {
	return r.g.Get().(rotateInfo)
}

func (r *rotateInfoGenerator) AddCallback(fn func(rotateInfo)) {
	f := func(val any) {
		fn(val.(rotateInfo))
	}
	r.g.AddCallback(f)
}

func (r *rotateInfoGenerator) AddCallbackWithCtx(fn func(context.Context, rotateInfo)) {
	f := func(ctx context.Context, val any) {
		fn(ctx, val.(rotateInfo))
	}
	r.g.AddCallbackWithCtx(f)
}

func (r *rotateInfoGenerator) Stop() {
	r.g.Stop()
}

type rotateRule struct {
	Span       time.Duration
	SuffixFunc func() string
}

// AddRotateRule 添加自定义时间分割规则
//
// 默认存在的规则：
//
// no: 不分割
//
// 1min: 每分钟分割一次，文件名格式：2006-01-02_1504'
//
// 5min: 每五分钟分割一次，文件名格式：2006-01-02_15xx
//
// 10min: 每十分钟分割一次，文件名格式：2006-01-02_15xx
//
// 15min: 每十五分钟分割一次，文件名格式：2006-01-02_15xx
//
// 30min: 每三十分钟分割一次，文件名格式：2006-01-02_15xx
//
// hour: 每小时分割一次，文件名格式：2006-01-02_15
//
// day: 每天分割一次，文件名格式：2006-01-02
func AddRotateRule(name string, span time.Duration, fn func() string) error {
	if _, ok := defaultRotateRule[name]; ok {
		return errors.New("rule already exists")
	}
	defaultRotateRule[name] = &rotateRule{
		Span:       span,
		SuffixFunc: fn,
	}
	return nil
}

var defaultRotateRule = map[string]*rotateRule{
	"no": {
		Span:       0,
		SuffixFunc: func() string { return "" },
	},
	"1min": {
		Span:       time.Minute,
		SuffixFunc: func() string { return "." + nowFunc().Format("2006-01-02_1504") },
	},
	"5min": {
		Span: time.Minute * 5,
		SuffixFunc: func() string {
			now := nowFunc()
			return "." + now.Format("2006-01-02_15") + fmt.Sprintf("%02d", now.Minute()/5*5)
		},
	},
	"10min": {
		Span: time.Minute * 10,
		SuffixFunc: func() string {
			now := nowFunc()
			return "." + now.Format("2006-01-02_15") + fmt.Sprintf("%02d", now.Minute()/10*10)
		},
	},
	"15min": {
		Span: time.Minute * 15,
		SuffixFunc: func() string {
			now := nowFunc()
			return "." + now.Format("2006-01-02_15") + fmt.Sprintf("%02d", now.Minute()/15*15)
		},
	},
	"30min": {
		Span: time.Minute * 30,
		SuffixFunc: func() string {
			now := nowFunc()
			return "." + now.Format("2006-01-02_15") + fmt.Sprintf("%02d", now.Minute()/30*30)
		},
	},
	"hour": {
		Span:       time.Hour,
		SuffixFunc: func() string { return "." + nowFunc().Format("2006-01-02_15") },
	},
	"day": {
		Span:       time.Hour * 24,
		SuffixFunc: func() string { return "." + nowFunc().Format("2006-01-02") },
	},
}
