package MiddleWare

import (
	"context"

	"github.com/gogf/gf/v2/net/ghttp"
)

type CtxKey string

var CustomCtxKey = CtxKey("custom_ctx")

type ContextUser struct {
	UserID       string  `json:"user_id"`
	UserName     string  `json:"user_name"`
	UserNickname string  `json:"user_nickname"`
	UserType     string  `json:"user_type"`
	Phone        string  `json:"phone"`
	OrgID        string  `json:"org_id"`
	RoleIDs      []int64 `json:"role_ids"`

	Token string `json:"-"`
}

type IContext interface {
	Init(r *ghttp.Request, info *ContextUser)
	GetBearerToken(ctx context.Context) (string, error)
	SetUserInfo(r *ghttp.Request, info *ContextUser)
	GetUserID(ctx context.Context) (string, error)
	GetUserName(ctx context.Context) (string, error)
	GetUserNickname(ctx context.Context) (string, error)
	GetUserType(ctx context.Context) (string, error)
	GetPhone(ctx context.Context) (string, error)
	GetOrgID(ctx context.Context) (string, error)
	GetRoleIDs(ctx context.Context) ([]int64, error)
}

type IAuth interface {
	// 解析令牌并将结果注入请求上下文，如果解析失败，则返回错误
	Auth(r *ghttp.Request)
}
