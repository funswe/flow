package flow

import (
	"net/http"
	"net/url"
	"github.com/julienschmidt/httprouter"
)

type Context struct {
	app    *Application
	req    *request
	res    *response
	params httprouter.Params
}

func NewContext(app *Application, w http.ResponseWriter, r *http.Request, params httprouter.Params) *Context {
	req := newRequest(app, r)
	res := newResponse(app, w, r)
	return &Context{req: req, app: app, res: res, params: params}
}

func (c *Context) GetHeaders() map[string][]string {
	return c.req.getHeaders()
}

func (c *Context) GetHeader(key string) string {
	return c.req.getHeader(key)
}

func (c *Context) GetUri() string {
	return c.req.getUri()
}

func (c *Context) GetHost() string {
	return c.req.getHost()
}

func (c *Context) GetProtocol() string {
	return c.req.getProtocol()
}

func (c *Context) IsSecure() bool {
	return c.req.isSecure()
}

func (c *Context) GetOrigin() string {
	return c.req.getOrigin()
}

func (c *Context) GetHref() string {
	return c.req.getHref()
}

func (c *Context) GetMethod() string {
	return c.req.getMethod()
}

func (c *Context) GetQuery() url.Values {
	return c.req.getQuery()
}

func (c *Context) GetQuerystring() string {
	return c.req.getQuerystring()
}

func (c *Context) GetHostname() string {
	return c.req.getHostname()
}

func (c *Context) GetLength() int {
	return c.req.getLength()
}

func (c *Context) SetHeader(key, value string) *Context {
	c.res.setHeader(key, value)
	return c
}

func (c *Context) SetStatus(code int) *Context {
	c.res.setStatus(code)
	return c
}

func (c *Context) SetLength(length int) *Context {
	c.res.setLength(length)
	return c
}

func (c *Context) Redirect(url string, code int) {
	c.res.redirect(url, code)
}

func (c *Context) Download(filePath string) {
	c.res.download(filePath)
}

func (c *Context) JsonResponse(data map[string]interface{}) *Context {
	c.res.jsonResponse(data)
	return c
}
