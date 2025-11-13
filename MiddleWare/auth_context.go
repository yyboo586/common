package MiddleWare

import (
	"context"
	"errors"

	"github.com/gogf/gf/v2/net/ghttp"
)

// Init 初始化上下文对象指针到上下文对象中，以便后续的请求流程中可以修改。
func ContextInit(r *ghttp.Request, info *ContextUser) {
	r.SetCtxVar(CustomCtxKey, info)
}

func GetContextUser(ctx context.Context) (*ContextUser, error) {
	value := ctx.Value(CustomCtxKey)
	if value == nil {
		return nil, errors.New("no custom data found")
	}
	return value.(*ContextUser), nil
}
