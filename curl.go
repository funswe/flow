package flow

import (
	"github.com/funswe/flow/utils/json"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

// CurlConfig 定义httpclient配置
type CurlConfig struct {
	Timeout time.Duration     // 请求的超时时间，单位秒
	Headers map[string]string // 统一请求的头信息
}

// 返回默认的httpclient配置
func defCurlConfig() *CurlConfig {
	return &CurlConfig{
		Timeout: 10 * time.Second, // 默认10秒超时时间
	}
}

// CurlResult 定义返回的结果
type CurlResult struct {
	*resty.Response
}

// Parse 将返回的结果定义到给定的对象里
func (cr *CurlResult) Parse(v interface{}) error {
	return json.Unmarshal(cr.Body(), v)
}

// Curl 定义httpclient对象
type Curl struct {
	app    *Application
	client *resty.Client
}

func (c *Curl) Get(url string, data map[string]string, headers map[string]string) (*CurlResult, error) {
	c.app.Logger.Debug("curl request start", zap.String("method", "get"),
		zap.String("url", url), zap.Any("data", data), zap.Any("headers", headers))
	r := c.client.R().SetHeaders(c.app.curlConfig.Headers)
	if data != nil {
		r.SetQueryParams(data)
	}
	if len(headers) > 0 {
		r.SetHeaders(headers)
	}
	res, err := r.Get(url)
	if err != nil {
		c.app.Logger.Error("curl request end", zap.Error(err))
		return nil, err
	}
	showBody := false
	if strings.HasPrefix(res.Header().Get(HttpHeaderContentType), "application/json") || strings.HasPrefix(res.Header().Get(HttpHeaderContentType), "text") {
		showBody = true
	}
	if showBody {
		c.app.Logger.Debug("curl request end", zap.Int("StatusCode", res.StatusCode()),
			zap.String("CostTime", res.Time().String()), zap.String("body", res.String()))
	} else {
		c.app.Logger.Debug("curl request end", zap.Int("StatusCode", res.StatusCode()),
			zap.String("CostTime", res.Time().String()))
	}
	return &CurlResult{res}, nil
}

func (c *Curl) Post(url string, data interface{}, headers map[string]string) (*CurlResult, error) {
	c.app.Logger.Debug("curl request start", zap.String("method", "post"),
		zap.String("url", url), zap.Any("data", data), zap.Any("headers", headers))
	r := c.client.R().SetHeaders(c.app.curlConfig.Headers)
	if data != nil {
		r.SetBody(data)
	}
	if len(headers) > 0 {
		r.SetHeaders(headers)
	}
	res, err := r.Post(url)
	if err != nil {
		c.app.Logger.Error("curl request end", zap.Error(err))
		return nil, err
	}
	showBody := false
	if strings.HasPrefix(res.Header().Get(HttpHeaderContentType), "application/json") || strings.HasPrefix(res.Header().Get(HttpHeaderContentType), "text") {
		showBody = true
	}
	if showBody {
		c.app.Logger.Debug("curl request end", zap.Int("StatusCode", res.StatusCode()),
			zap.String("CostTime", res.Time().String()), zap.String("body", res.String()))
	} else {
		c.app.Logger.Debug("curl request end", zap.Int("StatusCode", res.StatusCode()),
			zap.String("CostTime", res.Time().String()))
	}
	return &CurlResult{res}, nil
}

// 初始化httpclient对象
func initCurl(app *Application) {
	if app.curlConfig == nil {
		return
	}
	app.Curl = &Curl{
		app: app,
		client: resty.NewWithClient(&http.Client{
			Timeout: app.curlConfig.Timeout,
		}),
	}
	app.Logger.Info("curl server started", zap.Any("config", app.curlConfig))
}
