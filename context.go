package flow

import (
	"errors"
	"gorm.io/gorm"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/funswe/flow/log"
	"github.com/funswe/flow/utils/json"
	"github.com/julienschmidt/httprouter"
)

type Context struct {
	req    *request
	res    *response
	Logger *log.Logger
	params map[string]interface{}
	app    *Application
}

func newContext(w http.ResponseWriter, r *http.Request, params httprouter.Params, reqId int64, app *Application) *Context {
	req := newRequest(r, reqId, app)
	res := newResponse(w, req, app)
	r.ParseForm()
	mapParams := make(map[string]interface{})
	if len(params) > 0 {
		for i := range params {
			mapParams[params[i].Key] = params[i].Value
		}
	}
	for k := range r.Form {
		mapParams[k] = r.FormValue(k)
	}
	// 如果是json请求，解析json数据，如果form参数和json参数相同，json参数覆盖form参数
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
	ctxLogger := logFactory.Create(map[string]interface{}{
		"reqId": req.id,
		"ua":    req.getUserAgent(),
	})
	return &Context{req: req, res: res, params: mapParams, Logger: ctxLogger, app: app}
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

func (c *Context) Parse(object interface{}) error {
	body, err := json.Marshal(c.params)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, object)
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

func (c *Context) GetUserAgent() string {
	return c.req.getUserAgent()
}

func (c *Context) GetStatusCode() int {
	return c.res.getStatusCode()
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

func (c *Context) Json(data map[string]interface{}) {
	c.res.json(data)
}

func (c *Context) Body(body string) {
	c.res.text(body)
}

func (c *Context) Render(tmpFile string, data map[string]interface{}) {
	c.res.render(tmpFile, data)
}

func (c *Context) GetApp() *Application {
	return c.app
}

func (c *Context) DB() *gorm.DB {
	if c.app.db == nil {
		panic(errors.New("no db server available"))
	}
	return c.app.db
}
