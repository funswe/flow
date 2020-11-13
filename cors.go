package flow

type CorsConfig struct {
	Enable         bool
	AllowOrigin    string
	AllowedHeaders string
	AllowedMethod  string
}

func defCorsConfig() *CorsConfig {
	return &CorsConfig{
		Enable:        false,
		AllowOrigin:   "*",
		AllowedMethod: "GET, POST, HEAD, OPTIONS, PUT, PATCH, DELETE, TRACE",
	}
}
