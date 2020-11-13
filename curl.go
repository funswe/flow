package flow

import (
	"github.com/funswe/flow/utils/json"
	"github.com/go-resty/resty/v2"
	"github.com/google/go-querystring/query"
	"net/http"
	"time"
)

type CurlConfig struct {
	Enable  bool
	Timeout time.Duration
	Headers map[string]string
}

func defCurlConfig() *CurlConfig {
	return &CurlConfig{
		Enable:  false,
		Timeout: 10 * time.Second,
	}
}

type CurlResult string

func (cr CurlResult) Parse(v interface{}) error {
	return json.Unmarshal([]byte(string(cr)), v)
}

func (cr CurlResult) Raw(v interface{}) string {
	return string(cr)
}

type Curl struct {
	app    *Application
	client *resty.Client
}

func defCurl() *Curl {
	return &Curl{}
}

func (c *Curl) Get(url string, data interface{}, headers map[string]string) (CurlResult, error) {
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
		return CurlResult(""), err
	}
	logFactory.Debugf("curl request end, StatusCode: %d, CostTime: %s, body: %s", res.StatusCode(), res.Time(), res.String())
	return CurlResult(res.String()), nil
}

func (c *Curl) Post(url string, data interface{}, headers map[string]string) (CurlResult, error) {
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
		return CurlResult(""), err
	}
	logFactory.Debugf("curl request end, StatusCode: %d, CostTime: %s, body: %s", res.StatusCode(), res.Time(), res.String())
	return CurlResult(res.String()), nil
}

func initCurl(app *Application) {
	if app.curlConfig != nil && app.curlConfig.Enable {
		app.curl.client = resty.NewWithClient(&http.Client{
			Timeout: app.curlConfig.Timeout,
		})
		app.curl.app = app
		logFactory.Info("curl server init ok")
	}
}
