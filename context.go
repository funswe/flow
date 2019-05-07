package flow

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

type Context struct {
	app    *Application
	req    *request
	res    *response
	params map[string]interface{}
}

func newContext(app *Application, w http.ResponseWriter, r *http.Request, params map[string]interface{}) *Context {
	req := newRequest(app, r)
	res := newResponse(app, w, r)
	return &Context{req: req, app: app, res: res, params: params}
}

func (c *Context) GetParam(key string) (value string) {
	return c.GetParamDefault(key, "")
}

func (c *Context) GetParamDefault(key, defaultValue string) (value string) {
	switch jv := c.params[key].(type) {
	case string:
		value = jv
	case int:
		value = strconv.Itoa(jv)
	case int32:
		value = strconv.Itoa(int(jv))
	case int64:
		value = strconv.Itoa(int(jv))
	case float64:
		value = strconv.FormatFloat(jv, 'f', -1, 64)
	case float32:
		value = strconv.FormatFloat(float64(jv), 'f', -1, 32)
	default:
		value = ""
	}
	if len(value) == 0 {
		return defaultValue
	}
	return
}

func (c *Context) ParseStructure(object interface{}) {
	body, err := json.Marshal(c.params)
	if err != nil {
		return
	}
	json.Unmarshal(body, object)
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

func (c *Context) JsonResponse(data map[string]interface{}) {
	c.res.jsonResponse(data)
}

func (c *Context) Body(body string) {
	c.res.textResponse(body)
}
