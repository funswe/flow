package flow

// 定义跨域配置
type CorsConfig struct {
	Enable         bool // 是否开始跨域支持
	AllowOrigin    string
	AllowedHeaders string
	AllowedMethods string
}

// 返回默认的跨域配置
func defCorsConfig() *CorsConfig {
	return &CorsConfig{
		Enable:         false,
		AllowOrigin:    "*",
		AllowedMethods: "GET, POST, HEAD, OPTIONS, PUT, PATCH, DELETE, TRACE",
	}
}
