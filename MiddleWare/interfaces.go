package MiddleWare

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
