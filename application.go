package flow

import (
	"flag"
	"github.com/funswe/flow/log"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"path/filepath"
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
	appName  string
	proxy    bool
	address  string
	viewPath string
	logPath  string
)

func init() {
	flag.StringVar(&appName, "app-name", "flow", "set app name")
	flag.BoolVar(&proxy, "proxy", false, "set proxy mode")
	flag.StringVar(&address, "address", "localhost:12345", "set listen address")
	path, _ := filepath.Abs(".")
	flag.StringVar(&viewPath, "view-path", filepath.Join(path, "views"), "set view path")
	flag.StringVar(&logPath, "log-path", filepath.Join(path, "logs"), "set log path")
	flag.Parse()
}

type Next func()

type Middleware func(ctx *Context, next Next)

type Handler func(ctx *Context)

type Application struct {
	appName    string
	proxy      bool
	address    string
	viewPath   string
	logPath    string
	log        *log.Log
	middleware []Middleware
	router     *httprouter.Router
}

func New() *Application {
	log := log.New(logPath, "flow.log")
	log = log.Create(map[string]interface{}{
		"proxy":    proxy,
		"address":  address,
		"viewPath": viewPath,
		"logPath":  logPath,
	})
	log.Infoln("start params: ")
	return &Application{
		proxy:    proxy,
		address:  address,
		viewPath: viewPath,
		logPath:  logPath,
		log:      log,
		router:   httprouter.New(),
	}
}

func (app *Application) SetProxy(proxy bool) *Application {
	app.proxy = proxy
	return app
}

func (app *Application) SetAddress(address string) *Application {
	app.address = address
	return app
}

func (app *Application) SetViewPath(viewPath string) *Application {
	app.viewPath = viewPath
	return app
}

func (app *Application) Run() error {
	return http.ListenAndServe(app.address, app.router)
}

func (app *Application) Use(m Middleware) *Application {
	app.middleware = append(app.middleware, m)
	return app
}

func (app *Application) GetProxy() bool {
	return app.proxy
}

func (app *Application) GetViewPath() string {
	return app.viewPath
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
		ctx := newContext(app, w, r, params)
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
