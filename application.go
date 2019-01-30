package flow

import (
	"net/http"
	"github.com/julienschmidt/httprouter"
)

type Middleware func(ctx *Context)

type Handler func(ctx *Context)

type Application struct {
	proxy      bool
	middleware []Middleware
	router     *httprouter.Router
}

func New() *Application {
	return &Application{
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

func (app *Application) dispatch(ctx *Context, index int) {
	if index >= len(app.middleware) {
		return
	}
	app.middleware[index](ctx)
}

func (app *Application) handle(handler Handler) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		r.ParseForm()
		for k := range r.Form {
			params = append(params, httprouter.Param{
				Key:   k,
				Value: r.FormValue(k),
			})
		}
		ctx := NewContext(app, w, r, params)
		for _, middleware := range app.middleware {
			if middleware != nil {
				middleware(ctx)
			}
		}
		handler(ctx)
	}
}

func (app *Application) GET(path string, handler Handler) {
	app.router.Handle("GET", path, app.handle(handler))
}

func (app *Application) POST(path string, handler Handler) {
	app.router.Handle("POST", path, app.handle(handler))
}
