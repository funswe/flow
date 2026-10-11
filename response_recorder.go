package flow

import "net/http"

// ResponseRecorder 包装http.ResponseWriter，在透传响应的同时记录状态码与响应体，
// 供中间件在next()之后读取（访问日志、响应断言等场景）。
//
// maxBodyBytes>0时只缓存前maxBodyBytes个字节，超出部分置Truncated为true且不再缓存；
// maxBodyBytes<=0表示不限制缓存长度。缓存的始终是响应体的副本，不会影响真实响应。
type ResponseRecorder struct {
	http.ResponseWriter
	status       int
	body         []byte
	maxBodyBytes int
	Truncated    bool
}

// NewResponseRecorder 基于原writer创建响应记录器
func NewResponseRecorder(w http.ResponseWriter, maxBodyBytes int) *ResponseRecorder {
	return &ResponseRecorder{ResponseWriter: w, maxBodyBytes: maxBodyBytes}
}

// WriteHeader 记录首个状态码并透传，语义与http.ResponseWriter一致
func (r *ResponseRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

// Write 缓存响应体副本并透传，隐式状态码按http语义记为200
func (r *ResponseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	remain := len(data)
	if r.maxBodyBytes > 0 {
		if len(r.body) >= r.maxBodyBytes {
			remain = 0
		} else if left := r.maxBodyBytes - len(r.body); left < remain {
			remain = left
		}
	}
	if remain > 0 {
		r.body = append(r.body, data[:remain]...)
	}
	if remain < len(data) {
		r.Truncated = true
	}
	return r.ResponseWriter.Write(data)
}

// Status 返回已记录的状态码，未写入过任何响应时为0
func (r *ResponseRecorder) Status() int {
	return r.status
}

// Body 返回已缓存的响应体
func (r *ResponseRecorder) Body() []byte {
	return r.body
}

// Reset 清空已记录的状态码与响应体，便于复用
func (r *ResponseRecorder) Reset() {
	r.status = 0
	r.body = r.body[:0]
	r.Truncated = false
}
