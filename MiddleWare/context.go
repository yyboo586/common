package MiddleWare

import (
	"context"
	"errors"
	"sync"

	"github.com/gogf/gf/v2/net/ghttp"
)

var (
	customContextOnce     sync.Once
	customContextInstance IContext
)

type cuxtomContext struct {
}

func NewCustomContext() IContext {
	customContextOnce.Do(func() {
		customContextInstance = &cuxtomContext{}
	})
	return customContextInstance
}

// Init 初始化上下文对象指针到上下文对象中，以便后续的请求流程中可以修改。
func (s *cuxtomContext) Init(r *ghttp.Request, info *ContextUser) {
	r.SetCtxVar(CustomCtxKey, info)
}

func (s *cuxtomContext) GetBearerToken(ctx context.Context) (string, error) {
	userInfo, err := s.getUserInfo(ctx)
	if err != nil {
		return "", err
	}
	return userInfo.Token, nil
}

func (s *cuxtomContext) SetUserInfo(r *ghttp.Request, info *ContextUser) {
	r.SetCtxVar(CustomCtxKey, info)
}

func (s *cuxtomContext) IsLogin(ctx context.Context) bool {
	value := ctx.Value(CustomCtxKey)
	return value != nil
}

func (s *cuxtomContext) getUserInfo(ctx context.Context) (*ContextUser, error) {
	value := ctx.Value(CustomCtxKey)
	if value == nil {
		return nil, errors.New("user info not found")
	}
	return value.(*ContextUser), nil
}

func (s *cuxtomContext) GetUserID(ctx context.Context) (string, error) {
	userInfo, err := s.getUserInfo(ctx)
	if err != nil {
		return "", err
	}
	return userInfo.UserID, nil
}

func (s *cuxtomContext) GetUserName(ctx context.Context) (string, error) {
	userInfo, err := s.getUserInfo(ctx)
	if err != nil {
		return "", err
	}
	return userInfo.UserName, nil
}

func (s *cuxtomContext) GetUserNickname(ctx context.Context) (string, error) {
	userInfo, err := s.getUserInfo(ctx)
	if err != nil {
		return "", err
	}
	return userInfo.UserNickname, nil
}

func (s *cuxtomContext) GetUserType(ctx context.Context) (string, error) {
	userInfo, err := s.getUserInfo(ctx)
	if err != nil {
		return "", err
	}
	return userInfo.UserType, nil
}

func (s *cuxtomContext) GetPhone(ctx context.Context) (string, error) {
	userInfo, err := s.getUserInfo(ctx)
	if err != nil {
		return "", err
	}
	return userInfo.Phone, nil
}

func (s *cuxtomContext) GetOrgID(ctx context.Context) (string, error) {
	userInfo, err := s.getUserInfo(ctx)
	if err != nil {
		return "", err
	}
	return userInfo.OrgID, nil
}

func (s *cuxtomContext) GetRoleIDs(ctx context.Context) ([]int64, error) {
	userInfo, err := s.getUserInfo(ctx)
	if err != nil {
		return nil, err
	}
	return userInfo.RoleIDs, nil
}
