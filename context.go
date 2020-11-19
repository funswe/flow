package flow

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/funswe/flow/log"
	"github.com/funswe/flow/utils/json"
	"github.com/julienschmidt/httprouter"
)

// 定义请求上下文对象
type Context struct {
	req    *request               // 请求封装的request对象
	res    *response              // 请求封装的response对象
	Logger *log.Logger            // 上下文的logger对象，打印日志会自动带上请求的相关参数
	params map[string]interface{} // 请求的参数，包括POST，GET和路由的参数
	app    *Application           // 服务的APP对象
	Orm    *Orm                   // 数据库操作对象，引用app的orm对象
	Redis  *RedisClient           // redis操作对象，引用app的redis对象
	Curl   *Curl                  // httpclient操作对象，引用app的curl对象
	Jwt    *Jwt                   // JWT操作对象，引用app的jwt对象
}

// 返回一个新的context对象
func newContext(w http.ResponseWriter, r *http.Request, params httprouter.Params, reqId int64, app *Application) *Context {
	// 封装请求的request对象
	req := newRequest(r, reqId, app)
	// 封装请求的response对象
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
	// 定义上下文的logger对象，打印的时候带上请求的ID和ua
	ctxLogger := logFactory.Create(map[string]interface{}{
		"reqId": req.id,
		"ua":    req.getUserAgent(),
	})
	return &Context{req: req, res: res, params: mapParams, Logger: ctxLogger, app: app, Orm: app.orm, Redis: app.redis, Curl: app.curl, Jwt: app.jwt}
}

// 获取请求的参数
func (c *Context) GetParam(key string) (value string) {
	return c.GetParamDefault(key, "")
}

// 获取请求的参数，可以设置默认值
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

// 解析请求的参数，将参数赋值到给定的对象里
func (c *Context) Parse(object interface{}) error {
	if object == nil {
		return errors.New("object can not be nil")
	}
	t := reflect.TypeOf(object)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	fieldNum := t.NumField()
	for i := 0; i < fieldNum; i++ {
		field := t.Field(i)
		flowTag := field.Tag.Get("flow")
		if len(flowTag) == 0 {
			continue
		}
		ss := strings.Split(flowTag, ";")
		for _, tag := range ss {
			kv := strings.Split(tag, ":")
			if kv[0] == "required" && kv[1] == "true" {
				if _, ok := c.params[field.Name]; !ok {
					showName := field.Name
					jsonTag := field.Tag.Get("json")
					if len(jsonTag) > 0 {
						jsonName := strings.Split(jsonTag, ",")
						showName = jsonName[0]
					}
					return errors.New(fmt.Sprintf("required param `%s` is nil", showName))
				}
				break
			}
		}
	}
	body, err := json.Marshal(c.params)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, object)
}

// 获取请求的所有头信息
func (c *Context) GetHeaders() map[string][]string {
	return c.req.getHeaders()
}

// 获取请求的头信息
func (c *Context) GetHeader(key string) string {
	return c.req.getHeader(key)
}

// 获取请求的URI
func (c *Context) GetUri() string {
	return c.req.getUri()
}

// 获取请求的HOST信息
func (c *Context) GetHost() string {
	return c.req.getHost()
}

// 获取请求的协议，http or https
func (c *Context) GetProtocol() string {
	return c.req.getProtocol()
}

// 判断是不是https请求
func (c *Context) IsSecure() bool {
	return c.req.isSecure()
}

// 获取请求的地址
func (c *Context) GetOrigin() string {
	return c.req.getOrigin()
}

// 获取请求的完整链接
func (c *Context) GetHref() string {
	return c.req.getHref()
}

// 获取请求的方法，如GET,POST
func (c *Context) GetMethod() string {
	return c.req.getMethod()
}

// 获取请求的query参数，map格式
func (c *Context) GetQuery() url.Values {
	return c.req.getQuery()
}

// 获取请求的querystring
func (c *Context) GetQuerystring() string {
	return c.req.getQuerystring()
}

// 获取请求的hostname
func (c *Context) GetHostname() string {
	return c.req.getHostname()
}

// 获取请求的内容长度
func (c *Context) GetLength() int {
	return c.req.getLength()
}

// 获取请求的ua
func (c *Context) GetUserAgent() string {
	return c.req.getUserAgent()
}

// 获取返回的http状态码
func (c *Context) GetStatusCode() int {
	return c.res.getStatusCode()
}

// 设置返回的头信息
func (c *Context) SetHeader(key, value string) *Context {
	c.res.setHeader(key, value)
	return c
}

// 设置返回的http状态码
func (c *Context) SetStatus(code int) *Context {
	c.res.setStatus(code)
	return c
}

// 设置返回体的长度
func (c *Context) SetLength(length int) *Context {
	c.res.setLength(length)
	return c
}

// 设置返回的重定向地址
func (c *Context) Redirect(url string, code int) {
	c.res.redirect(url, code)
}

// 下载文件
func (c *Context) Download(filePath string) {
	c.res.download(filePath)
}

// 返回json数据
func (c *Context) Json(data map[string]interface{}) {
	c.res.json(data)
}

// 返回文本数据
func (c *Context) Body(body string) {
	c.res.text(body)
}

// 返回服务端渲染文本信息
func (c *Context) Render(tmpFile string, data map[string]interface{}) {
	c.res.render(tmpFile, data)
}

// 获取app对象
func (c *Context) GetApp() *Application {
	return c.app
}
