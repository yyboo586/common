package MiddleWare

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/yyboo586/common/httpUtils"
)

var (
	addr   string
	client httpUtils.HTTPClient
)

func init() {
	addr = g.Cfg().MustGet(context.Background(), "gfToken.introspectAddr").String()
	client = httpUtils.NewHTTPClient()
}

func Auth(r *ghttp.Request) {
	ctx := r.GetCtx()

	excludePaths := g.Cfg().MustGet(ctx, "gfToken.excludePaths").Strings()
	// 如果请求路径在排除路径列表中，则直接放行
	if slices.Contains(excludePaths, r.URL.Path) {
		r.Middleware.Next()
		return
	}

	contextInfo, err := introspectToken(r)
	// 如果解析令牌失败，退出后续所有中间件，并返回401错误
	if err != nil {
		r.Response.WriteJson(DefaultResponse{
			Code:    http.StatusUnauthorized,
			Message: err.Error(),
		})
		r.ExitAll()
	}

	// 将用户信息初始化到上下文
	ContextInit(r, contextInfo)
	r.Middleware.Next()
}

func introspectToken(r *ghttp.Request) (out *ContextUser, err error) {
	tokenStr := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if tokenStr == "" {
		return nil, gerror.New("token is required")
	}

	header := map[string]interface{}{
		"Authorization": "Bearer " + tokenStr,
	}

	status, respBody, err := client.POST(r.Context(), addr, header, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, gerror.New("token is invalid")
	}

	var resp HttpResp
	err = json.Unmarshal(respBody, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, gerror.New(resp.Message)
	}

	userID, _ := resp.Data.(map[string]interface{})["user_id"].(string)
	userName, _ := resp.Data.(map[string]interface{})["user_name"].(string)
	userNickname, _ := resp.Data.(map[string]interface{})["user_nickname"].(string)
	userType, _ := resp.Data.(map[string]interface{})["user_type"].(string)
	phone, _ := resp.Data.(map[string]interface{})["user_phone"].(string)
	orgID, _ := resp.Data.(map[string]interface{})["org_id"].(string)
	roleIDs, _ := resp.Data.(map[string]interface{})["role_ids"].([]int64)

	out = &ContextUser{
		UserID:       userID,
		UserName:     userName,
		UserNickname: userNickname,
		UserType:     userType,
		Phone:        phone,
		OrgID:        orgID,
		RoleIDs:      roleIDs,

		Token: tokenStr,
	}
	return out, nil
}

type HttpResp struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}
