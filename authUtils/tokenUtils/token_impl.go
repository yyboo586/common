package tokenUtils

import (
	"context"
	"errors"
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/glog"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	BearerPrefix = "Bearer "
)

type Token struct {
	// 访问令牌过期时间 默认1天.单位:秒
	AccessTokenTimeout time.Duration
	// 刷新机制超时时间 默认5天.单位:秒
	RefreshTimeout time.Duration

	// 拦截排除地址
	ExcludePaths g.SliceStr

	// jwt
	signer *JwtSign

	logger *glog.Logger
}

// 生成token
func (t *Token) Generate(ctx context.Context, data interface{}) (token string, err error) {
	token, err = t.signer.CreateToken(CustomClaims{
		data,
		jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),                           // 颁发时间
			NotBefore: jwt.NewNumericDate(time.Now()),                           // 生效开始时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.AccessTokenTimeout)), // 失效截止时间
			ID:        uuid.New().String(),
		},
	})
	if err != nil {
		t.logger.Debugf(ctx, err.Error())
		return "", gerror.Wrap(err, "生成Token失败")
	}

	return token, nil
}

// 解析token (只验证格式并不验证过期)
func (t *Token) Parse(r *ghttp.Request) (*CustomClaims, error) {
	// 从请求中获取token
	token := t.GetTokenFromRequest(r)
	if token == "" {
		t.logger.Debugf(r.GetCtx(), "Token为空")
		return nil, gerror.New("Token为空")
	}

	// 解析token
	customClaims, err := t.signer.ParseToken(token)
	if err != nil {
		t.logger.Debugf(r.GetCtx(), err.Error())
		return nil, err
	}

	return customClaims, nil
}

// 刷新令牌
func (t *Token) Refresh(ctx context.Context, oldToken string) (newToken string, err error) {
	customClaims, err := t.signer.ParseToken(oldToken)
	if err != nil {
		t.logger.Debugf(ctx, err.Error())
		return "", err
	}

	if customClaims.IssuedAt.Add(t.RefreshTimeout).Before(time.Now()) {
		t.logger.Debugf(ctx, "刷新令牌已过期")
		return "", errors.New("refresh token is expired")
	}

	customClaims.IssuedAt = jwt.NewNumericDate(time.Now())
	customClaims.NotBefore = jwt.NewNumericDate(time.Now())
	customClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(t.AccessTokenTimeout))

	return t.signer.CreateToken(*customClaims)
}

func (t *Token) GetTokenFromRequest(r *ghttp.Request) (token string) {
	// 请求头获取
	n := len(BearerPrefix)
	auth := r.Header.Get("Authorization")
	if len(auth) >= n && auth[:n] == BearerPrefix {
		return auth[n:]
	}
	// 查询参数
	if q := r.Get("token"); !q.IsEmpty() {
		return q.String()
	}
	// Cookies
	if c := r.Cookie.Get("token"); !c.IsEmpty() {
		return c.String()
	}
	return
}
