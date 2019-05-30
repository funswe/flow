package flow

import (
	"github.com/funswe/flow/log"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"
)

const (
	HTTP_METHOD_GET     = "GET"
	HTTP_METHOD_HEAD    = "HEAD"
	HTTP_METHOD_OPTIONS = "OPTIONS"
	HTTP_METHOD_POST    = "POST"
	HTTP_METHOD_PUT     = "PUT"
	HTTP_METHOD_PATCH   = "PATCH"
	HTTP_METHOD_DELETE  = "DELETE"
)

var (
	logFactory     *log.Logger
	logger         *log.Logger
	reqId          int64
	rc             = make(chan int64)
	appName        string
	proxy          bool
	address        string
	viewPath       string
	logPath        string
	loggerLevel    string
	middleware     []Middleware
	router         = httprouter.New()
	panicHandler   PanicHandler
	notFoundHandle NotFoundHandle
)

type Next func()

type PanicHandler func(http.ResponseWriter, *http.Request, interface{})

type NotFoundHandle func(w http.ResponseWriter, r *http.Request)

func (f NotFoundHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

type Middleware func(ctx *Context, next Next)

type Handler func(ctx *Context)

func defaultErrorHandle() PanicHandler {
	return func(w http.ResponseWriter, r *http.Request, err interface{}) {
		logFactory.Error(err, "\n", string(debug.Stack()))
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(500)
		w.Write([]byte("unknown server error"))
	}
}

func defaultNotFoundHandle() NotFoundHandle {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
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

func defaultLoggerLevel() string {
	return "debug"
}

func Run() error {
	if len(appName) == 0 {
		appName = defaultAppName()
	}
	if !proxy {
		proxy = defaultProxy()
	}
	if len(address) == 0 {
		address = defaultAddress()
	}
	if len(viewPath) == 0 {
		viewPath = defaultViewPath()
	}
	if len(logPath) == 0 {
		logPath = defaultLogPath()
	}
	if len(loggerLevel) == 0 {
		loggerLevel = defaultLoggerLevel()
	}
	logFactory = log.New(logPath, appName+".log", loggerLevel)
	logger := logFactory.Create(map[string]interface{}{
		"appName":  appName,
		"proxy":    proxy,
		"address":  address,
		"viewPath": viewPath,
		"logPath":  logPath,
	})
	logger.Info("start params")
	// 启动一个独立的携程处理请求ID的递增
	go func() {
		for {
			reqId++
			rc <- reqId
		}
	}()
	if panicHandler == nil {
		panicHandler = defaultErrorHandle()
	}
	if notFoundHandle == nil {
		notFoundHandle = defaultNotFoundHandle()
	}
	router.PanicHandler = panicHandler
	router.NotFound = notFoundHandle
	middleware = append([]Middleware{func(ctx *Context, next Next) {
		start := time.Now().UnixNano()
		ctx.Logger.Debugf("request incoming, method: %s, uri: %s, host: %s, protocol: %s", ctx.GetMethod(), ctx.GetUri(), ctx.GetHost(), ctx.GetProtocol())
		next()
		cost := time.Now().UnixNano() - start
		ctx.Logger.Debugf("request finish, cost: %d ms, statusCode: %d", cost/1000000, ctx.GetStatusCode())
	}, func(ctx *Context, next Next) {
		ctx.SetHeader("X-Powered-By", "Flow")
		next()
	}}, middleware...)
	return http.ListenAndServe(address, router)
}

func Use(m Middleware) {
	middleware = append(middleware, m)
}

func SetAppName(a string) {
	if len(a) <= 0 {
		return
	}
	appName = a
}

func SetProxy(p bool) {
	proxy = p
}

func SetAddress(addr string) {
	if len(addr) <= 0 {
		return
	}
	address = addr
}

func SetViewPath(vp string) {
	if len(vp) <= 0 {
		return
	}
	viewPath = vp
}

func SetLogPath(lp string) {
	if len(lp) <= 0 {
		return
	}
	logPath = lp
}

func SetLoggerLevel(lv string) {
	if len(lv) <= 0 {
		return
	}
	loggerLevel = lv
}

func SetPanicHandler(ph PanicHandler) {
	panicHandler = ph
}

func SetNotFoundHandle(nfh NotFoundHandle) {
	notFoundHandle = nfh
}

func dispatch(ctx *Context, index int, handler Handler) Next {
	if index >= len(middleware) {
		return func() {
			handler(ctx)
		}
	}
	return func() {
		middleware[index](ctx, dispatch(ctx, index+1, handler))
	}
}

func handle(handler Handler) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		rwr := rwresponse{w, 200}
		ctx := newContext(&rwr, r, params, <-rc)
		dispatch(ctx, 0, handler)()
	}
}

func GET(path string, handler Handler) {
	router.Handle(HTTP_METHOD_GET, path, handle(handler))
}

func HEAD(path string, handler Handler) {
	router.Handle(HTTP_METHOD_HEAD, path, handle(handler))
}

func OPTIONS(path string, handler Handler) {
	router.Handle(HTTP_METHOD_OPTIONS, path, handle(handler))
}

func POST(path string, handler Handler) {
	router.Handle(HTTP_METHOD_POST, path, handle(handler))
}

func PUT(path string, handler Handler) {
	router.Handle(HTTP_METHOD_PUT, path, handle(handler))
}

func PATCH(path string, handler Handler) {
	router.Handle(HTTP_METHOD_PATCH, path, handle(handler))
}

func DELETE(path string, handler Handler) {
	router.Handle(HTTP_METHOD_DELETE, path, handle(handler))
}

func StaticFiles(prefix, path string) {
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	prefix = prefix + "*filepath"
	router.ServeFiles(prefix, http.Dir(path))
}

func ALL(path string, handler Handler) {
	GET(path, handler)
	HEAD(path, handler)
	OPTIONS(path, handler)
	POST(path, handler)
	PUT(path, handler)
	PATCH(path, handler)
	DELETE(path, handler)
}
