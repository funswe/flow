package flow

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestJwt(t *testing.T, timeout time.Duration) *Jwt {
	t.Helper()
	original := app.jwtConfig
	app.jwtConfig = &JwtConfig{Timeout: timeout, SecretKey: "unit-test-secret"}
	t.Cleanup(func() { app.jwtConfig = original })

	originalLogger := app.Logger
	app.Logger = zap.NewNop()
	t.Cleanup(func() { app.Logger = originalLogger })

	initJwt(app)
	if app.Jwt == nil {
		t.Fatal("initJwt应初始化Jwt对象")
	}
	return app.Jwt
}

// Sign 签发的token可以被 Valid 验证并取回原始数据
func TestJwtSignAndValid(t *testing.T) {
	jwt := newTestJwt(t, time.Hour)

	token, err := jwt.Sign(map[string]interface{}{"uid": "1001", "role": "admin"})
	if err != nil {
		t.Fatalf("签发token失败：%v", err)
	}
	if len(token) == 0 {
		t.Fatal("token不应为空")
	}

	data, err := jwt.Valid(token)
	if err != nil {
		t.Fatalf("验证token失败：%v", err)
	}
	if data["uid"] != "1001" || data["role"] != "admin" {
		t.Fatalf("token内容不符：%v", data)
	}
}

// 过期token应验证失败
func TestJwtExpiredToken(t *testing.T) {
	jwt := newTestJwt(t, -time.Minute)

	token, err := jwt.Sign(map[string]interface{}{"uid": "1001"})
	if err != nil {
		t.Fatalf("签发token失败：%v", err)
	}
	if _, err := jwt.Valid(token); err == nil {
		t.Fatal("过期token应验证失败")
	}
}

// 换密钥后旧token应验证失败
func TestJwtWrongSecret(t *testing.T) {
	jwt := newTestJwt(t, time.Hour)
	token, err := jwt.Sign(map[string]interface{}{"uid": "1001"})
	if err != nil {
		t.Fatalf("签发token失败：%v", err)
	}

	app.jwtConfig.SecretKey = "another-secret"
	if _, err := jwt.Valid(token); err == nil {
		t.Fatal("密钥变更后token应验证失败")
	}
}

// 非法token字符串应返回错误而不是panic
func TestJwtInvalidToken(t *testing.T) {
	jwt := newTestJwt(t, time.Hour)
	if _, err := jwt.Valid("not-a-jwt"); err == nil {
		t.Fatal("非法token应返回错误")
	}
}
