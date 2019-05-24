package flow

import (
	"github.com/funswe/flow/log"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"path/filepath"
	"runtime/debug"
	"strings"
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
	logFactory *log.Logger
	reqId      int64
	reqChan    = make(chan int64)
)

type Next func()

type PanicHandler func(http.ResponseWriter, *http.Request, interface{})

type NotFoundHandle func(w http.ResponseWriter, r *http.Request)

func (f NotFoundHandle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f(w, r)
}

type Middleware func(ctx *Context, next Next)

type Handler func(ctx *Context)

type Application struct {
	appName    string
	proxy      bool
	address    string
	viewPath   string
	logPath    string
	logger     *log.Logger
	middleware []Middleware
	router     *httprouter.Router
}

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
	return "localhost:12345"
}

func defaultViewPath() string {
	path, _ := filepath.Abs(".")
	return filepath.Join(path, "views")
}

func defaultLogPath() string {
	path, _ := filepath.Abs(".")
	return filepath.Join(path, "logs")
}

func New() *Application {
	router := httprouter.New()
	router.PanicHandler = defaultErrorHandle()
	router.NotFound = defaultNotFoundHandle()
	return &Application{
		appName:  defaultAppName(),
		proxy:    defaultProxy(),
		address:  defaultAddress(),
		viewPath: defaultViewPath(),
		logPath:  defaultLogPath(),
		router:   router,
	}
}

func (app *Application) init() {
	logFactory = log.New(app.GetLogPath(), app.GetAppName()+".log")
	app.logger = logFactory.Create(map[string]interface{}{
		"appName":  app.GetAppName(),
		"proxy":    app.GetProxy(),
		"address":  app.GetAddress(),
		"viewPath": app.GetViewPath(),
		"logPath":  app.GetLogPath(),
	})
	app.logger.Info("start params")
	// 启动一个独立的携程处理请求ID的递增
	go func() {
		for {
			reqId++
			reqChan <- reqId
		}
	}()
}

func (app *Application) Run() error {
	app.init()
	return http.ListenAndServe(app.address, app.router)
}

func (app *Application) Use(m Middleware) *Application {
	app.middleware = append(app.middleware, m)
	return app
}

func (app *Application) GetAppName() string {
	return app.appName
}

func (app *Application) SetAppName(appName string) *Application {
	if len(appName) <= 0 {
		return app
	}
	app.appName = appName
	return app
}

func (app *Application) GetProxy() bool {
	return app.proxy
}

func (app *Application) SetProxy(proxy bool) *Application {
	app.proxy = proxy
	return app
}

func (app *Application) GetAddress() string {
	return app.address
}

func (app *Application) SetAddress(address string) *Application {
	if len(address) <= 0 {
		return app
	}
	app.address = address
	return app
}

func (app *Application) GetViewPath() string {
	return app.viewPath
}

func (app *Application) SetViewPath(viewPath string) *Application {
	if len(viewPath) <= 0 {
		return app
	}
	app.viewPath = viewPath
	return app
}

func (app *Application) GetLogPath() string {
	return app.logPath
}

func (app *Application) SetLogPath(logPath string) *Application {
	if len(logPath) <= 0 {
		return app
	}
	app.logPath = logPath
	return app
}

func (app *Application) SetPanicHandler(panicHandler PanicHandler) *Application {
	app.router.PanicHandler = panicHandler
	return app
}

func (app *Application) dispatch(ctx *Context, index int, handler Handler) Next {
	if index >= len(app.middleware) {
		return func() {
			handler(ctx)
		}
	}
	return func() {
		app.middleware[index](ctx, app.dispatch(ctx, index+1, handler))
	}
}

func (app *Application) handle(handler Handler) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := newContext(app, w, r, params, <-reqChan)
		app.dispatch(ctx, 0, handler)()
	}
}

func (app *Application) GET(path string, handler Handler) *Application {
	app.router.Handle(HTTP_METHOD_GET, path, app.handle(handler))
	return app
}

func (app *Application) HEAD(path string, handler Handler) *Application {
	app.router.Handle(HTTP_METHOD_HEAD, path, app.handle(handler))
	return app
}

func (app *Application) OPTIONS(path string, handler Handler) *Application {
	app.router.Handle(HTTP_METHOD_OPTIONS, path, app.handle(handler))
	return app
}

func (app *Application) POST(path string, handler Handler) *Application {
	app.router.Handle(HTTP_METHOD_POST, path, app.handle(handler))
	return app
}

func (app *Application) PUT(path string, handler Handler) *Application {
	app.router.Handle(HTTP_METHOD_PUT, path, app.handle(handler))
	return app
}

func (app *Application) PATCH(path string, handler Handler) *Application {
	app.router.Handle(HTTP_METHOD_PATCH, path, app.handle(handler))
	return app
}

func (app *Application) DELETE(path string, handler Handler) *Application {
	app.router.Handle(HTTP_METHOD_DELETE, path, app.handle(handler))
	return app
}

func (app *Application) StaticFiles(prefix, path string) *Application {
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix = prefix + "/"
	}
	prefix = prefix + "*filepath"
	app.router.ServeFiles(prefix, http.Dir(path))
	return app
}

func (app *Application) ALL(path string, handler Handler) *Application {
	rHandle := app.handle(handler)
	app.router.Handle(HTTP_METHOD_GET, path, rHandle)
	app.router.Handle(HTTP_METHOD_HEAD, path, rHandle)
	app.router.Handle(HTTP_METHOD_OPTIONS, path, rHandle)
	app.router.Handle(HTTP_METHOD_POST, path, rHandle)
	app.router.Handle(HTTP_METHOD_PUT, path, rHandle)
	app.router.Handle(HTTP_METHOD_PATCH, path, rHandle)
	app.router.Handle(HTTP_METHOD_DELETE, path, rHandle)
	return app
}
