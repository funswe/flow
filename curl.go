package flow

import (
	"github.com/funswe/flow/utils/json"
	"github.com/go-resty/resty/v2"
	"github.com/google/go-querystring/query"
	"net/http"
	"time"
)

// 定义httpclient配置
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

// 定义返回的结果
type CurlResult struct {
	*resty.Response
}

// 将返回的结果定义到给定的对象里
func (cr *CurlResult) Parse(v interface{}) error {
	return json.Unmarshal(cr.Body(), v)
}

// 定义httpclient对象
type Curl struct {
	app    *Application
	client *resty.Client
}

func defCurl() *Curl {
	return &Curl{}
}

func (c *Curl) Get(url string, data interface{}, headers map[string]string) (*CurlResult, error) {
	logFactory.Debugf("curl request start, method: get, url: %s, data: %v, headers: %v", url, data, headers)
	r := c.client.R().SetHeaders(c.app.curlConfig.Headers)
	if data != nil {
		v, _ := query.Values(data)
		r.SetQueryParamsFromValues(v)
	}
	if len(headers) > 0 {
		r.SetHeaders(headers)
	}
	res, err := r.Get(url)
	if err != nil {
		logFactory.Errorf("curl request end, error: %s", err.Error())
		return nil, err
	}
	logFactory.Debugf("curl request end, StatusCode: %d, CostTime: %s, body: %s", res.StatusCode(), res.Time(), res.String())
	return &CurlResult{res}, nil
}

func (c *Curl) Post(url string, data interface{}, headers map[string]string) (*CurlResult, error) {
	logFactory.Debugf("curl request start, method: post, url: %s, data: %v, headers: %v", url, data, headers)
	r := c.client.R().SetHeaders(c.app.curlConfig.Headers)
	if data != nil {
		r.SetBody(data)
	}
	if len(headers) > 0 {
		r.SetHeaders(headers)
	}
	res, err := r.Post(url)
	if err != nil {
		logFactory.Errorf("curl request end, error: %s", err.Error())
		return nil, err
	}
	logFactory.Debugf("curl request end, StatusCode: %d, CostTime: %s, body: %s", res.StatusCode(), res.Time(), res.String())
	return &CurlResult{res}, nil
}

// 初始化httpclient对象
func initCurl(app *Application) {
	app.curl.client = resty.NewWithClient(&http.Client{
		Timeout: app.curlConfig.Timeout,
	})
	app.curl.app = app
	logFactory.Info("curl server init ok")
}
