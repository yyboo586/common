package tokenUtils

import (
	"context"
	"encoding/json"
	"time"

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

// TokenPair 令牌对，包含访问令牌和刷新令牌
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type IToken interface {
	Generate(ctx context.Context, data interface{}) (pair *TokenPair, err error)
	Parse(r *ghttp.Request) (*CustomClaims, bool, error)
	GetTokenFromRequest(r *ghttp.Request) (token string)
	Refresh(ctx context.Context, refreshToken string) (pair *TokenPair, err error)

	RevokeToken(ctx context.Context, token string) error
	RevokeDeviceToken(ctx context.Context, deviceID string) error
	RevokeUserToken(ctx context.Context, userID string) error
}

// 令牌持久化数据
type TokenEntity struct {
	ID             string `orm:"id"`
	UserID         string `orm:"user_id"`
	DeviceID       string `orm:"device_id"`
	Content        string `orm:"content"`
	TokenType      string `orm:"token_type"`
	RefreshTokenID string `orm:"refresh_token_id"`
	IsActive       int    `orm:"is_active"`
	ExpireTime     int64  `orm:"expire_time"`
	CreateTime     int64  `orm:"create_time"`
	UpdateTime     int64  `orm:"update_time"`
}

type TokenInfo struct {
	ID         string
	UserID     string
	DeviceID   string
	Content    interface{}
	IsActive   bool
	ExpireTime time.Time
	CreateTime time.Time
	UpdateTime time.Time
}

func ConvertTokenEntityToModel(in *TokenEntity) (out *TokenInfo) {
	out = &TokenInfo{
		ID:         in.ID,
		UserID:     in.UserID,
		DeviceID:   in.DeviceID,
		IsActive:   in.IsActive == 1,
		ExpireTime: time.Time{},
		CreateTime: time.Time{},
		UpdateTime: time.Time{},
	}

	if in.Content != "" {
		_ = json.Unmarshal([]byte(in.Content), &out.Content)
	}
	if in.ExpireTime > 0 {
		out.ExpireTime = time.Unix(in.ExpireTime, 0)
	}
	if in.CreateTime > 0 {
		out.CreateTime = time.Unix(in.CreateTime, 0)
	}
	if in.UpdateTime > 0 {
		out.UpdateTime = time.Unix(in.UpdateTime, 0)
	}

	return out
}
