package tokenUtils

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/golang-jwt/jwt/v5"
)

const (
	ErrTokenInvalidText   string = "无效的Token"
	ErrTokenMalFormedText string = "Token格式不正确"
)

type CustomClaims struct {
	Data interface{}
	jwt.RegisteredClaims
}

type IToken interface {
	Generate(ctx context.Context, data interface{}) (token string, err error)
	Parse(r *ghttp.Request) (*CustomClaims, error)
	Refresh(ctx context.Context, oldToken string) (newToken string, err error)
	GetTokenFromRequest(r *ghttp.Request) (token string)
}
