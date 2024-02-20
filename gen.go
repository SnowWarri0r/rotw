package rotw

import (
	"context"
	"sync"
	"time"
)

// Generator 按照一定的时间间隔生产数据，并将生成的数据传递给注册的回调函数
//
// 生成的数据也可通过 Get() 直接获取
//
// 通过 Stop() 停止生产数据，并 cancel 掉上下文，使用者可以自行处理上下文
type Generator interface {
	// Get 获取最近一次生成的数据
	Get() any
	// AddCallback 添加回调函数，当生成数据时，会调用回调函数
	AddCallback(func(any))
	// AddCallbackWithCtx 添加回调函数，当生成数据时，会调用回调函数
	AddCallbackWithCtx(func(context.Context, any))
	// Stop 停止生产数据，并 cancel 掉上下文，使用者可以自行处理上下文
	Stop()
}

// NewGenerator 创建 Generator，并启动定时器
func NewGenerator(span time.Duration, genFn func() any) Generator {
	ctx, cancel := context.WithCancel(context.Background())
	p := &generator{
		ctx:    ctx,
		cancel: cancel,
		span:   span,
		genFn:  genFn,
	}
	p.start()
	return p
}

// generator 按照一定的时间间隔生成数据，并将生成的数据传递给注册的回调函数
type generator struct {
	// 控制生命周期
	ctx         context.Context
	cancel      func()
	span        time.Duration
	callbacks   []func(context.Context, any)
	genFn       func() any
	timer       *time.Timer
	mux         sync.Mutex
	lastTrigger int64
	lastProduct any
}

func (g *generator) AddCallback(callback func(any)) {
	g.mux.Lock()
	defer g.mux.Unlock()
	fn := func(ctx context.Context, product any) {
		callback(product)
	}
	g.callbacks = append(g.callbacks, fn)
}

func (g *generator) AddCallbackWithCtx(callback func(context.Context, any)) {
	g.mux.Lock()
	defer g.mux.Unlock()
	g.callbacks = append(g.callbacks, callback)
}

// Get 获取最近一次生成的数据
func (g *generator) Get() any {
	g.mux.Lock()
	defer g.mux.Unlock()
	return g.lastProduct
}

func (g *generator) start() {
	// 启动就生成一次数据
	_ = g.gen()
	// 周期为0，则不启动定时器
	if g.span.Nanoseconds() == 0 {
		return
	}
	// 启动定时器，定时生成数据
	g.timer = time.AfterFunc(g.next(), func() {
		val := g.gen()
		g.notify(val)
		g.timer.Reset(g.next())
	})
	g.lastTrigger = nowFunc().Unix()
	go g.doCheck()
}

// Stop 停止生产数据，并 cancel 掉上下文，使用者可以自行处理上下文
func (g *generator) Stop() {
	g.cancel()
	g.mux.Lock()
	defer g.mux.Unlock()
	if g.timer == nil {
		return
	}
	g.timer.Stop()
}

func (g *generator) doCheck() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-g.ctx.Done():
			return
		case <-ticker.C:
			g.check()
		}
	}
}

// check 检测系统时间是否变化，如果变化则重置定时器
func (g *generator) check() {
	now := nowFunc().Unix()
	defer func() {
		g.lastTrigger = now
	}()
	gap := now - g.lastTrigger
	// 检查每秒都执行一次，所以如果间隔小于2秒，则不需要重置定时器
	if gap == 1 || gap == 2 {
		return
	}
	g.mux.Lock()
	defer g.mux.Unlock()
	g.timer.Stop()
	g.timer.Reset(g.next())
}

// gen 生成数据
func (g *generator) gen() any {
	g.mux.Lock()
	defer g.mux.Unlock()
	g.lastProduct = g.genFn()
	return g.lastProduct
}

// notify 通知所有注册的回调函数
func (g *generator) notify(val any) {
	for _, callback := range g.callbacks {
		go callback(g.ctx, val)
	}
}

// next 计算下一次触发的时间
func (g *generator) next() time.Duration {
	_, offset := nowFunc().Zone()
	// Unix时间戳没有时区信息，所以需要手动加上时区偏移
	localTs := time.Duration(nowFunc().Unix()+int64(offset)) * time.Second
	ret := g.span - localTs%g.span
	return ret
}
