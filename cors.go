package flow

// CorsConfig 定义跨域配置
type CorsConfig struct {
	AllowOrigin    string
	AllowedHeaders string
	AllowedMethods string
}

// 返回默认的跨域配置
func defCorsConfig() *CorsConfig {
	return &CorsConfig{
		AllowOrigin:    "*",
		AllowedMethods: "GET, POST, HEAD, OPTIONS, PUT, PATCH, DELETE, TRACE",
	}
}
