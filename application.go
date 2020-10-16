package flow

import (
	"net/http"
	"os"
	"time"

	"github.com/funswe/flow/log"
	"github.com/funswe/flow/utils/files"
)

type Application struct {
	logger     *log.Logger
	reqId      int64
	rc         chan int64
	appName    string
	proxy      bool
	address    string
	viewPath   string
	staticPath string
	middleware []Middleware
}

func (app *Application) run() error {
	if len(app.staticPath) != 0 {
		if !files.PathExists(app.staticPath) {
			os.MkdirAll(app.staticPath, os.ModePerm)
		}
	}
	// 启动一个独立的携程处理请求ID的递增
	go func() {
		for {
			app.reqId++
			app.rc <- app.reqId
		}
	}()
	app.middleware = append([]Middleware{func(ctx *Context, next Next) {
		start := time.Now().UnixNano()
		ctx.Logger.Debugf("request incoming, method: %s, uri: %s, host: %s, protocol: %s", ctx.GetMethod(), ctx.GetUri(), ctx.GetHost(), ctx.GetProtocol())
		next()
		cost := time.Now().UnixNano() - start
		ctx.Logger.Debugf("request completed, cost: %d ms, statusCode: %d", cost/1000000, ctx.GetStatusCode())
	}, func(ctx *Context, next Next) {
		ctx.SetHeader(HttpHeaderXPoweredBy, app.appName)
		next()
	}}, app.middleware...)
	return http.ListenAndServe(app.address, router)
}

func (app *Application) use(m Middleware) {
	app.middleware = append(app.middleware, m)
}

func (app *Application) setAppName(a string) {
	if len(a) <= 0 {
		return
	}
	app.appName = a
}

func (app *Application) setProxy(p bool) {
	app.proxy = p
}

func (app *Application) setAddress(addr string) {
	if len(addr) <= 0 {
		return
	}
	app.address = addr
}

func (app *Application) setViewPath(vp string) {
	if len(vp) <= 0 {
		return
	}
	app.viewPath = vp
}
