package flow

type CorsConfig struct {
	Enable         bool
	AllowOrigin    string
	AllowedHeaders string
	AllowedMethods string
}

func defCorsConfig() *CorsConfig {
	return &CorsConfig{
		Enable:         false,
		AllowOrigin:    "*",
		AllowedMethods: "GET, POST, HEAD, OPTIONS, PUT, PATCH, DELETE, TRACE",
	}
}
