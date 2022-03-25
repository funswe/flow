package flow

import (
	"errors"
	"fmt"
	"github.com/funswe/flow/log"
	"github.com/funswe/flow/utils/json"
	"github.com/julienschmidt/httprouter"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
)

const defaultMultipartMemory = 32 << 20 // 32 MB

type FieldValidateError struct {
	Type  string
	Value interface{}
	Field string
	Param string
}

func (e *FieldValidateError) Error() string {
	switch e.Type {
	case "required":
		return fmt.Sprintf("required param `%s` is nil", e.Param)
	default:
		return fmt.Sprintf("required param `%s` is nil", e.Param)
	}
}

// 定义请求上下文对象
type Context struct {
	req    *request               // 请求封装的request对象
	res    *response              // 请求封装的response对象
	mu     sync.RWMutex           // 互斥锁，用于data map
	data   map[string]interface{} // 用于保存用户定义的数据
	params map[string]interface{} // 请求的参数，包括POST，GET和路由的参数
	app    *Application           // 服务的APP对象
	Logger *log.Logger            // 上下文的logger对象，打印日志会自动带上请求的相关参数
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
	// 判断是不是上传文件
	if strings.HasPrefix(req.getHeader("Content-Type"), "multipart/form-data") {
		r.ParseMultipartForm(defaultMultipartMemory)
	} else {
		r.ParseForm()
	}
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

// 保存用户设置的数据
func (c *Context) SetData(key string, value interface{}) {
	c.mu.Lock()
	if c.data == nil {
		c.data = make(map[string]interface{})
	}
	c.data[key] = value
	c.mu.Unlock()
}

// 获取用户保存的值
func (c *Context) GetData(key string) (value interface{}, exists bool) {
	c.mu.RLock()
	value, exists = c.data[key]
	c.mu.RUnlock()
	return
}

// 获取请求的string参数，如果参数类型不是string，则会转换成string，只支持基本类型转换
func (c *Context) GetStringParam(key string) (value string) {
	if val, ok := c.params[key]; ok && val != nil {
		switch jv := c.params[key].(type) {
		case string:
			value = jv
		case int:
			value = strconv.Itoa(jv)
		case int32:
			value = strconv.Itoa(int(jv))
		case int64:
			value = strconv.Itoa(int(jv))
		case float32:
			value = strconv.FormatFloat(float64(jv), 'f', -1, 32)
		case float64:
			value = strconv.FormatFloat(jv, 'f', -1, 64)
		case bool:
			value = ""
			if jv {
				value = "1"
			}
		default:
			value = ""
		}
	}
	return
}

// GetStringParam方法的带默认值
func (c *Context) GetStringParamDefault(key string, def string) (value string) {
	val := c.GetStringParam(key)
	if len(val) == 0 {
		return def
	}
	return val
}

// 获取请求的int参数，如果参数类型不是int，则会转换成int，只支持基本类型转换
func (c *Context) GetIntParam(key string) (value int) {
	if val, ok := c.params[key]; ok && val != nil {
		switch jv := c.params[key].(type) {
		case string:
			value, _ = strconv.Atoi(jv)
		case int:
			value = jv
		case int32:
			value = int(jv)
		case int64:
			value = int(jv)
		case float32:
			value = int(jv)
		case float64:
			value = int(jv)
		case bool:
			value = 0
			if jv {
				value = 1
			}
		default:
			value = 0
		}
	}
	return
}

// GetIntParam方法的带默认值
func (c *Context) GetIntParamDefault(key string, def int) (value int) {
	val := c.GetIntParam(key)
	if val == 0 {
		return def
	}
	return val
}

// 获取请求的int64参数，如果参数类型不是int64，则会转换成int64，只支持基本类型转换
func (c *Context) GetInt64Param(key string) (value int64) {
	if val, ok := c.params[key]; ok && val != nil {
		switch jv := c.params[key].(type) {
		case string:
			v, _ := strconv.Atoi(jv)
			value = int64(v)
		case int:
			value = int64(jv)
		case int32:
			value = int64(jv)
		case int64:
			value = jv
		case float32:
			value = int64(jv)
		case float64:
			value = int64(jv)
		case bool:
			value = 0
			if jv {
				value = 1
			}
		default:
			value = 0
		}
	}
	return
}

// GetInt64Param方法的带默认值
func (c *Context) GetInt64ParamDefault(key string, def int64) (value int64) {
	val := c.GetInt64Param(key)
	if val == 0 {
		return def
	}
	return val
}

// 获取请求的float64参数，如果参数类型不是float64，则会转换成float64，只支持基本类型转换
func (c *Context) GetFloat64Param(key string) (value float64) {
	if val, ok := c.params[key]; ok && val != nil {
		switch jv := c.params[key].(type) {
		case string:
			v, _ := strconv.Atoi(jv)
			value = float64(v)
		case int:
			value = float64(jv)
		case int32:
			value = float64(jv)
		case int64:
			value = float64(jv)
		case float32:
			value = float64(jv)
		case float64:
			value = jv
		case bool:
			value = 0
			if jv {
				value = 1
			}
		default:
			value = 0
		}
	}
	return
}

// GetFloat64Param方法的带默认值
func (c *Context) GetFloat64ParamDefault(key string, def float64) (value float64) {
	val := c.GetFloat64Param(key)
	if val == 0 {
		return def
	}
	return val
}

// 获取请求的bool参数，如果参数类型不是bool，则会转换成bool，只支持基本类型转换
func (c *Context) GetBoolParam(key string) (value bool) {
	if val, ok := c.params[key]; ok && val != nil {
		switch jv := c.params[key].(type) {
		case string:
			v, _ := strconv.Atoi(jv)
			value = v > 0
		case int:
			value = jv > 0
		case int32:
			value = jv > 0
		case int64:
			value = jv > 0
		case float32:
			value = jv > 0
		case float64:
			value = jv > 0
		case bool:
			value = jv
		default:
			value = false
		}
	}
	return
}

// GetBoolParam方法的带默认值
func (c *Context) GetBoolParamDefault(key string, def bool) (value bool) {
	val := c.GetBoolParam(key)
	if !val {
		return def
	}
	return val
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
				showName := field.Name
				jsonTag := field.Tag.Get("json")
				if len(jsonTag) > 0 {
					jsonName := strings.Split(jsonTag, ",")
					showName = jsonName[0]
				}
				if _, ok := c.params[showName]; !ok {
					return &FieldValidateError{
						Type:  "required",
						Field: field.Name,
						Param: showName,
					}
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

func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	if c.req.req.MultipartForm == nil {
		if err := c.req.req.ParseMultipartForm(defaultMultipartMemory); err != nil {
			return nil, err
		}
	}
	f, fh, err := c.req.req.FormFile(name)
	if err != nil {
		return nil, err
	}
	f.Close()
	return fh, err
}

// 保存上传的文件到指定位置
func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string, flag int) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	out, err := os.OpenFile(dst, flag, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, src)
	return err
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

// 获取请求的客户端的IP
func (c *Context) GetClientIp() string {
	return c.req.getClientIp()
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

// 返回buffer数据
func (c *Context) Buffer(buffer []byte) {
	c.res.raw(buffer)
}

// 获取app对象
func (c *Context) GetApp() *Application {
	return c.app
}
