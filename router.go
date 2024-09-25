package flow

import (
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
	"net/http"
	"runtime/debug"
	"time"
)

var (
	router         = httprouter.New()        // 路由对象
	panicHandler   = defaultErrorHandle()    // 统一错误处理方法
	notFoundHandle = defaultNotFoundHandle() // 路由不存在处理方法
)

func init() {
	router.PanicHandler = panicHandler
	router.NotFound = notFoundHandle
}

type RouterGroup struct {
	middleware []Middleware
}

type Next func()

type PanicHandler func(http.ResponseWriter, *http.Request, interface{})

type NotFoundHandle func(w http.ResponseWriter, r *http.Request)

func (f NotFoundHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// Middleware 定义中间件接口
type Middleware func(ctx *Context, next Next)

// Handler 定义路由处理器
type Handler func(ctx *Context)

func dispatch(ctx *Context, index int, handler Handler, rg *RouterGroup) Next {
	if index >= len(rg.middleware) {
		return func() {
			handler(ctx)
		}
	}
	return func() {
		rg.middleware[index](ctx, dispatch(ctx, index+1, handler, rg))
	}
}

func handle(handler Handler, rg *RouterGroup) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := newContext(w, r, params, app)
		dispatch(ctx, 0, handler, rg)()
	}
}

func NewRouterGroup() *RouterGroup {
	rg := &RouterGroup{}
	// 添加默认的中间件
	rg.middleware = append([]Middleware{func(ctx *Context, next Next) {
		// 添加请求日志打印
		start := time.Now()
		ctx.Logger.Info("request incoming",
			zap.String("method", ctx.GetMethod()), zap.String("uri", ctx.GetUri()),
			zap.String("host", ctx.GetHost()), zap.String("protocol", ctx.GetProtocol()))
		next()
		ctx.Logger.Info("request completed",
			zap.String("cost", time.Since(start).Round(time.Millisecond).String()),
			zap.Int("statusCode", ctx.statusCode))
	}, func(ctx *Context, next Next) {
		ctx.SetHeader(HttpHeaderXPoweredBy, "flow")
		// 添加跨域支持
		ctx.SetHeader(HttpHeaderCorsOrigin, app.corsConfig.AllowOrigin)
		ctx.SetHeader(HttpHeaderCorsMethods, app.corsConfig.AllowedMethods)
		ctx.SetHeader(HttpHeaderCorsHeaders, app.corsConfig.AllowedHeaders)
		ctx.SetHeader(HttpHeaderCorsMaxAge, "172800")
		if ctx.GetMethod() == HttpMethodOptions {
			ctx.res.raw([]byte("true"))
			return
		}
		next()
	}}, rg.middleware...)
	return rg
}

// Use 添加中间件
func (rg *RouterGroup) Use(m Middleware) *RouterGroup {
	rg.middleware = append(rg.middleware, m)
	return rg
}

func (rg *RouterGroup) GET(path string, handler Handler) *RouterGroup {
	router.Handle(HttpMethodGet, path, handle(handler, rg))
	router.Handle(HttpMethodOptions, path, handle(handler, rg))
	return rg
}

func (rg *RouterGroup) HEAD(path string, handler Handler) *RouterGroup {
	router.Handle(HttpMethodHead, path, handle(handler, rg))
	router.Handle(HttpMethodOptions, path, handle(handler, rg))
	return rg
}

func (rg *RouterGroup) POST(path string, handler Handler) *RouterGroup {
	router.Handle(HttpMethodPost, path, handle(handler, rg))
	router.Handle(HttpMethodOptions, path, handle(handler, rg))
	return rg
}

func (rg *RouterGroup) PUT(path string, handler Handler) *RouterGroup {
	router.Handle(HttpMethodPut, path, handle(handler, rg))
	router.Handle(HttpMethodOptions, path, handle(handler, rg))
	return rg
}

func (rg *RouterGroup) PATCH(path string, handler Handler) *RouterGroup {
	router.Handle(HttpMethodPatch, path, handle(handler, rg))
	router.Handle(HttpMethodOptions, path, handle(handler, rg))
	return rg
}

func (rg *RouterGroup) DELETE(path string, handler Handler) *RouterGroup {
	router.Handle(HttpMethodDelete, path, handle(handler, rg))
	router.Handle(HttpMethodOptions, path, handle(handler, rg))
	return rg
}

func (rg *RouterGroup) ALL(path string, handler Handler) *RouterGroup {
	router.Handle(HttpMethodGet, path, handle(handler, rg))
	router.Handle(HttpMethodHead, path, handle(handler, rg))
	router.Handle(HttpMethodPost, path, handle(handler, rg))
	router.Handle(HttpMethodPut, path, handle(handler, rg))
	router.Handle(HttpMethodPatch, path, handle(handler, rg))
	router.Handle(HttpMethodDelete, path, handle(handler, rg))
	router.Handle(HttpMethodOptions, path, handle(handler, rg))
	return rg
}

func defaultErrorHandle() PanicHandler {
	return func(w http.ResponseWriter, r *http.Request, err interface{}) {
		w.Header().Set(HttpHeaderContentType, "text/plain; charset=utf-8")
		w.WriteHeader(500)
		if v, ok := err.(string); ok {
			_, _ = w.Write([]byte(v))
		} else if v, ok := err.(error); ok {
			_, _ = w.Write([]byte(v.Error()))
		} else {
			_, _ = w.Write([]byte("unknown server error"))
		}
	}
}

func defaultNotFoundHandle() NotFoundHandle {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HttpHeaderContentType, "text/plain; charset=utf-8")
		w.Header().Set(HttpHeaderXContentTypeOptions, "nosniff")
		w.WriteHeader(404)
		_, _ = w.Write([]byte("404 page not found"))
	}
}

func SetPanicHandler(ph PanicHandler) {
	if ph == nil {
		ph = defaultErrorHandle()
	}
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, err interface{}) {
		app.Logger.Error("error", zap.ByteString("Stack", debug.Stack()))
		ph(w, r, err)
	}
}

func SetNotFoundHandle(nfh NotFoundHandle) {
	if nfh == nil {
		nfh = defaultNotFoundHandle()
	}
	router.NotFound = nfh
}
