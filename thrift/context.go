package thrift

import (
	"context"
	"github.com/SkyAPM/go2sky"
	"github.com/timandy/routine"
)

var (
	threadLocal = routine.NewInheritableThreadLocal()
	//threadLocal = routine.NewThreadLocal()
)

type InvokeContext struct {
	entryCtx  context.Context
	span      go2sky.Span
	entrySpan go2sky.Span
}

func storeToContext(entryCtx context.Context, span, entrySpan go2sky.Span) {
	threadLocal.Set(&InvokeContext{
		entryCtx:  entryCtx,
		span:      span,
		entrySpan: entrySpan,
	})
}

func recoverFromContext() (context.Context, go2sky.Span, go2sky.Span) {
	if v := threadLocal.Get(); v != nil {
		ic := v.(*InvokeContext)
		return ic.entryCtx, ic.span, ic.entrySpan
	} else {
		return nil, nil, nil
	}
}
