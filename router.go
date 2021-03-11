package flow

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"runtime/debug"
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

type Next func()

type PanicHandler func(http.ResponseWriter, *http.Request, interface{})

type NotFoundHandle func(w http.ResponseWriter, r *http.Request)

func (f NotFoundHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

// 定义中间件接口
type Middleware func(ctx *Context, next Next)

// 定义路由处理器
type Handler func(ctx *Context)

func dispatch(ctx *Context, index int, handler Handler) Next {
	if index >= len(app.middleware) {
		return func() {
			handler(ctx)
		}
	}
	return func() {
		app.middleware[index](ctx, dispatch(ctx, index+1, handler))
	}
}

func handle(handler Handler) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		rwr := rwresponse{w, 200}
		ctx := newContext(&rwr, r, params, <-app.rc, app)
		dispatch(ctx, 0, handler)()
	}
}

func GET(path string, handler Handler) {
	router.Handle(HttpMethodGet, path, handle(handler))
	router.Handle(HttpMethodOptions, path, handle(handler))
}

func HEAD(path string, handler Handler) {
	router.Handle(HttpMethodHead, path, handle(handler))
	router.Handle(HttpMethodOptions, path, handle(handler))
}

func POST(path string, handler Handler) {
	router.Handle(HttpMethodPost, path, handle(handler))
	router.Handle(HttpMethodOptions, path, handle(handler))
}

func PUT(path string, handler Handler) {
	router.Handle(HttpMethodPut, path, handle(handler))
	router.Handle(HttpMethodOptions, path, handle(handler))
}

func PATCH(path string, handler Handler) {
	router.Handle(HttpMethodPatch, path, handle(handler))
	router.Handle(HttpMethodOptions, path, handle(handler))
}

func DELETE(path string, handler Handler) {
	router.Handle(HttpMethodDelete, path, handle(handler))
	router.Handle(HttpMethodOptions, path, handle(handler))
}

//func StaticFiles(prefix, path string) {
//	app.staticPath = path
//	if !strings.HasPrefix(prefix, "/") {
//		prefix = "/" + prefix
//	}
//	if !strings.HasSuffix(prefix, "/") {
//		prefix = prefix + "/"
//	}
//	prefix = prefix + "*filepath"
//	router.ServeFiles(prefix, http.Dir(path))
//}

func ALL(path string, handler Handler) {
	router.Handle(HttpMethodGet, path, handle(handler))
	router.Handle(HttpMethodHead, path, handle(handler))
	router.Handle(HttpMethodPost, path, handle(handler))
	router.Handle(HttpMethodPut, path, handle(handler))
	router.Handle(HttpMethodPatch, path, handle(handler))
	router.Handle(HttpMethodDelete, path, handle(handler))
	router.Handle(HttpMethodOptions, path, handle(handler))
}

func defaultErrorHandle() PanicHandler {
	return func(w http.ResponseWriter, r *http.Request, err interface{}) {
		w.Header().Set(HttpHeaderContentType, "text/plain; charset=utf-8")
		w.WriteHeader(500)
		if v, ok := err.(string); ok {
			w.Write([]byte(v))
		} else if v, ok := err.(error); ok {
			w.Write([]byte(v.Error()))
		} else {
			w.Write([]byte("unknown server error"))
		}
	}
}

func defaultNotFoundHandle() NotFoundHandle {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HttpHeaderContentType, "text/plain; charset=utf-8")
		w.Header().Set(HttpHeaderXContentTypeOptions, "nosniff")
		w.WriteHeader(404)
		w.Write([]byte("404 page not found"))
	}
}

func SetPanicHandler(ph PanicHandler) {
	if ph == nil {
		ph = defaultErrorHandle()
	}
	router.PanicHandler = func(w http.ResponseWriter, r *http.Request, err interface{}) {
		logFactory.Error(err, "\n", string(debug.Stack()))
		ph(w, r, err)
	}
}

func SetNotFoundHandle(nfh NotFoundHandle) {
	if nfh == nil {
		nfh = defaultNotFoundHandle()
	}
	router.NotFound = nfh
}
