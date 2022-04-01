package flow

import (
	"errors"
	"github.com/golang-jwt/jwt"
	"time"
)

// 定义JWT配置
type JwtConfig struct {
	Timeout   time.Duration // 请求的超时时间，单位小时
	SecretKey string        // 秘钥
}

// 返回默认的JWT配置
func defJwtConfig() *JwtConfig {
	return &JwtConfig{
		Timeout: 24 * time.Hour, // 默认24小时有效时间
	}
}

// 定义JWT对象
type Jwt struct {
	app *Application
}

func defJwt() *Jwt {
	return &Jwt{}
}

type Claims struct {
	jwt.StandardClaims
	Data map[string]interface{}
}

func (j *Jwt) Sign(data map[string]interface{}) (string, error) {
	claims := &Claims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(app.jwtConfig.Timeout).Unix(), // 过期时间，必须设置
			Issuer:    "flow",                                       // 可不必设置，也可以填充用户名，
		},
		Data: data,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) //生成token
	accessToken, err := token.SignedString([]byte(j.app.jwtConfig.SecretKey))
	if err != nil {
		return "", err
	}
	return accessToken, nil
}

func (j *Jwt) Valid(token string) (map[string]interface{}, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.app.jwtConfig.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := tokenClaims.Claims.(*Claims); ok && tokenClaims.Valid {
		return claims.Data, nil
	}
	return nil, errors.New("invalid token")
}

// 初始化JWT对象
func initJwt(app *Application) {
	app.Jwt.app = app
	logFactory.Info("jwt server init ok")
}
