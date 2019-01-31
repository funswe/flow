package flow

import (
	"net/http"
	"github.com/julienschmidt/httprouter"
	"strings"
	"io/ioutil"
	"encoding/json"
)

type Next func()

type Middleware func(ctx *Context, next Next)

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

func (app *Application) dispatch(ctx *Context, index int) Next {
	if index >= len(app.middleware) {
		return func() {}
	}
	return func() {
		app.middleware[index](ctx, app.dispatch(ctx, index+1))
	}
}

func (app *Application) handle(handler Handler) func(http.ResponseWriter, *http.Request, httprouter.Params) {
	return func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		r.ParseForm()
		mapParams := make(map[string]interface{})
		for k := range r.Form {
			mapParams[k] = r.FormValue(k)
		}
		// handle json request, json data replace form data
		if r.Body != nil && strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			result, err := ioutil.ReadAll(r.Body)
			if err == nil && len(result) > 0 {
				jsonMap := make(map[string]interface{})
				err = json.Unmarshal(result, &jsonMap)
				if err == nil {
					for k := range jsonMap {
						mapParams[k] = jsonMap[k]
					}
				}
			}
		}
		ctx := NewContext(app, w, r, mapParams)
		app.dispatch(ctx, 0)()
		handler(ctx)
	}
}

func (app *Application) GET(path string, handler Handler) {
	app.router.Handle("GET", path, app.handle(handler))
}

func (app *Application) HEAD(path string, handler Handler) {
	app.router.Handle("HEAD", path, app.handle(handler))
}

func (app *Application) OPTIONS(path string, handler Handler) {
	app.router.Handle("OPTIONS", path, app.handle(handler))
}

func (app *Application) POST(path string, handler Handler) {
	app.router.Handle("POST", path, app.handle(handler))
}

func (app *Application) PUT(path string, handler Handler) {
	app.router.Handle("PUT", path, app.handle(handler))
}

func (app *Application) PATCH(path string, handler Handler) {
	app.router.Handle("PATCH", path, app.handle(handler))
}

func (app *Application) DELETE(path string, handler Handler) {
	app.router.Handle("DELETE", path, app.handle(handler))
}

func (app *Application) ALL(path string, handler Handler) {
	rHandle := app.handle(handler)
	app.router.Handle("GET", path, rHandle)
	app.router.Handle("HEAD", path, rHandle)
	app.router.Handle("OPTIONS", path, rHandle)
	app.router.Handle("POST", path, rHandle)
	app.router.Handle("PUT", path, rHandle)
	app.router.Handle("PATCH", path, rHandle)
	app.router.Handle("DELETE", path, rHandle)
}
