package flow

import (
	"github.com/funswe/flow/log"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"path/filepath"
	"runtime/debug"
)

const (
	HttpMethodGet     = "GET"
	HttpMethodHead    = "HEAD"
	HttpMethodOptions = "OPTIONS"
	HttpMethodPost    = "POST"
	HttpMethodPut     = "PUT"
	HttpMethodPatch   = "PATCH"
	HttpMethodDelete  = "DELETE"
)

const (
	HttpHeaderContentType             = "Content-Type"
	HttpHeaderContentLength           = "Content-Length"
	HttpHeaderTransferEncoding        = "Transfer-Encoding"
	HttpHeaderContentDisposition      = "Content-Disposition"
	HttpHeaderContentTransferEncoding = "Content-Transfer-Encoding"
	HttpHeaderExpires                 = "Expires"
	HttpHeaderCacheControl            = "Cache-Control"
	HttpHeaderEtag                    = "Etag"
	HttpHeaderXForwardedHost          = "X-Forwarded-Host"
	HttpHeaderXForwardedProto         = "X-Forwarded-Proto"
	HttpHeaderIfModifiedSince         = "If-Modified-Since"
	HttpHeaderIfNoneMatch             = "If-None-Match"
	HttpHeaderLastModified            = "Last-Modified"
	HttpHeaderXContentTypeOptions     = "X-Content-Type-Options"
	HttpHeaderXPoweredBy              = "X-Powered-By"
)

var (
	logFactory     *log.Logger
	loggerLevel    = defaultLoggerLevel()
	logPath        = defaultLogPath()
	router         = httprouter.New()
	panicHandler   = defaultErrorHandle()
	notFoundHandle = defaultNotFoundHandle()
	app            = &Application{
		rc:         make(chan int64),
		appName:    defaultAppName(),
		proxy:      defaultProxy(),
		address:    defaultAddress(),
		viewPath:   defaultViewPath(),
		staticPath: defaultStaticPath(),
	}
)

type Next func()

type PanicHandler func(http.ResponseWriter, *http.Request, interface{})

type NotFoundHandle func(w http.ResponseWriter, r *http.Request)

func (f NotFoundHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

type Middleware func(ctx *Context, next Next)

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
}

func HEAD(path string, handler Handler) {
	router.Handle(HttpMethodHead, path, handle(handler))
}

func OPTIONS(path string, handler Handler) {
	router.Handle(HttpMethodOptions, path, handle(handler))
}

func POST(path string, handler Handler) {
	router.Handle(HttpMethodPost, path, handle(handler))
}

func PUT(path string, handler Handler) {
	router.Handle(HttpMethodPut, path, handle(handler))
}

func PATCH(path string, handler Handler) {
	router.Handle(HttpMethodPatch, path, handle(handler))
}

func DELETE(path string, handler Handler) {
	router.Handle(HttpMethodDelete, path, handle(handler))
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
	GET(path, handler)
	HEAD(path, handler)
	OPTIONS(path, handler)
	POST(path, handler)
	PUT(path, handler)
	PATCH(path, handler)
	DELETE(path, handler)
}

func defaultErrorHandle() PanicHandler {
	return func(w http.ResponseWriter, r *http.Request, err interface{}) {
		logFactory.Error(err, "\n", string(debug.Stack()))
		w.Header().Set(HttpHeaderContentType, "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("unknown server error"))
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

func defaultAppName() string {
	return "flow"
}

func defaultProxy() bool {
	return false
}

func defaultAddress() string {
	return "localhost:9505"
}

func defaultViewPath() string {
	path, _ := filepath.Abs(".")
	return filepath.Join(path, "views")
}

func defaultLogPath() string {
	path, _ := filepath.Abs(".")
	return filepath.Join(path, "logs")
}

func defaultStaticPath() string {
	path, _ := filepath.Abs(".")
	return filepath.Join(path, "resource")
}

func defaultLoggerLevel() string {
	return "debug"
}

func Use(m Middleware) {
	app.use(m)
}

func SetAppName(a string) {
	if len(a) <= 0 {
		a = defaultAppName()
	}
	app.setAppName(a)
}

func SetProxy(p bool) {
	app.setProxy(p)
}

func SetAddress(addr string) {
	if len(addr) <= 0 {
		addr = defaultAddress()
	}
	app.setAddress(addr)
}

func SetViewPath(vp string) {
	if len(vp) <= 0 {
		vp = defaultViewPath()
	}
	app.setViewPath(vp)
}

func SetLogPath(lp string) {
	if len(lp) <= 0 {
		lp = defaultLogPath()
	}
	logPath = lp
}

func SetLoggerLevel(lv string) {
	if len(lv) <= 0 {
		lv = defaultLogPath()
	}
	loggerLevel = lv
}

func SetPanicHandler(ph PanicHandler) {
	if ph == nil {
		ph = defaultErrorHandle()
	}
	panicHandler = ph
}

func SetNotFoundHandle(nfh NotFoundHandle) {
	if nfh == nil {
		nfh = defaultNotFoundHandle()
	}
	notFoundHandle = nfh
}

func Run() error {
	logFactory = log.New(logPath, app.appName+".log", loggerLevel)
	logger := logFactory.Create(map[string]interface{}{
		"appName":    app.appName,
		"proxy":      app.proxy,
		"address":    app.address,
		"viewPath":   app.viewPath,
		"logPath":    logPath,
		"staticPath": app.staticPath,
	})
	logger.Info("start params")
	// 启动一个独立的携程处理请求ID的递增
	router.PanicHandler = panicHandler
	router.NotFound = notFoundHandle
	return app.run()
}
