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
	notFoundHandle = defaultNotFoundHandle() // 路由不存在处理方法
	// appPanicHandler 应用配置的panic处理器（SetPanicHandler可替换）。
	// handler层的panic与中间件层的panic共用同一份，保证响应口径一致。
	appPanicHandler = defaultErrorHandle()
)

func init() {
	// httprouter这层只兜中间件自身的panic；handler的panic由dispatch最内层收口
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, err interface{}) {
		logPanicStack(app.Logger, debug.Stack())
		appPanicHandler(w, r, err)
	}
	router.NotFound = notFoundHandle
}

// logPanicStack 打印panic堆栈，便于与业务日志、访问日志关联排查
func logPanicStack(logger *zap.Logger, stack []byte) {
	if logger != nil {
		logger.Error("error", zap.ByteString("Stack", stack))
	}
}

// flowPanic 收口handler的panic：记录panic值与完整堆栈，再交由应用配置的处理器写出响应。
// 响应写到ctx的writer上（而非最初那个writer），这样中间件包装过的writer（如访问日志的
// ResponseRecorder）也能观察到这次响应；此处调用栈仍包含处理器帧，所以堆栈是完整的。
func flowPanic(ctx *Context, err interface{}) {
	stack := debug.Stack()
	ctx.panicValue = err
	ctx.panicStack = stack
	// 用ctx的logger带上请求上下文（reqId、ua等）打印，便于和访问日志对应
	logPanicStack(ctx.Logger, stack)
	appPanicHandler(ctx.res.res, ctx.req.req, err)
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
			// 最内层包住handler：panic在这里收口（记录堆栈、写出响应），不再向上抛。
			// 这样外层所有中间件都能正常走完，可以像普通响应一样观察到panic的结果。
			defer func() {
				if recovered := recover(); recovered != nil {
					flowPanic(ctx, recovered)
				}
			}()
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

// SetPanicHandler 设置应用自己的panic处理器（handler层与中间件层panic共用），
// 传nil恢复默认处理器。
func SetPanicHandler(ph PanicHandler) {
	if ph == nil {
		ph = defaultErrorHandle()
	}
	appPanicHandler = ph
}

func SetNotFoundHandle(nfh NotFoundHandle) {
	if nfh == nil {
		nfh = defaultNotFoundHandle()
	}
	router.NotFound = nfh
}
