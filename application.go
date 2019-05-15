package flow

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strings"
)

type Next func()

type Middleware func(ctx *Context, next Next)

type Handler func(ctx *Context)

type Application struct {
	proxy      bool
	middleware []Middleware
	router     *httprouter.Router
}

func New(proxy bool) *Application {
	return &Application{
		proxy:  proxy,
		router: httprouter.New(),
	}
}

func (app *Application) Run(addr string) error {
	return http.ListenAndServe(addr, app.router)
}

func (app *Application) Use(m Middleware) *Application {
	app.middleware = append(app.middleware, m)
	return app
}

func (app *Application) GetProxy() bool {
	return app.proxy
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
	app.router.Handle("GET", path, app.handle(handler))
	return app
}

func (app *Application) HEAD(path string, handler Handler) *Application {
	app.router.Handle("HEAD", path, app.handle(handler))
	return app
}

func (app *Application) OPTIONS(path string, handler Handler) *Application {
	app.router.Handle("OPTIONS", path, app.handle(handler))
	return app
}

func (app *Application) POST(path string, handler Handler) *Application {
	app.router.Handle("POST", path, app.handle(handler))
	return app
}

func (app *Application) PUT(path string, handler Handler) *Application {
	app.router.Handle("PUT", path, app.handle(handler))
	return app
}

func (app *Application) PATCH(path string, handler Handler) *Application {
	app.router.Handle("PATCH", path, app.handle(handler))
	return app
}

func (app *Application) DELETE(path string, handler Handler) *Application {
	app.router.Handle("DELETE", path, app.handle(handler))
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
	app.router.Handle("GET", path, rHandle)
	app.router.Handle("HEAD", path, rHandle)
	app.router.Handle("OPTIONS", path, rHandle)
	app.router.Handle("POST", path, rHandle)
	app.router.Handle("PUT", path, rHandle)
	app.router.Handle("PATCH", path, rHandle)
	app.router.Handle("DELETE", path, rHandle)
	return app
}
