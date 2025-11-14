package tokenUtils

import (
	"context"
	"encoding/json"
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
	store  *TokenStore
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

// 生成token
func (t *Token) Generate(ctx context.Context, data interface{}) (pair *TokenPair, err error) {
	// 生成访问令牌
	accessTokenJTI := uuid.New().String()
	accessClaims := CustomClaims{
		Data: data,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),                           // 颁发时间
			NotBefore: jwt.NewNumericDate(time.Now()),                           // 生效开始时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.AccessTokenTimeout)), // 失效截止时间
			ID:        accessTokenJTI,
		},
	}

	accessToken, err := t.signer.CreateToken(accessClaims)
	if err != nil {
		t.logger.Debugf(ctx, err.Error())
		return nil, gerror.Wrap(err, "生成访问令牌失败")
	}

	// 生成刷新令牌
	refreshTokenJTI := uuid.New().String()
	refreshClaims := CustomClaims{
		Data: data,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),                       // 颁发时间
			NotBefore: jwt.NewNumericDate(time.Now()),                       // 生效开始时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.RefreshTimeout)), // 失效截止时间（比访问令牌长）
			ID:        refreshTokenJTI,
		},
	}

	refreshToken, err := t.signer.CreateToken(refreshClaims)
	if err != nil {
		t.logger.Debugf(ctx, err.Error())
		return nil, gerror.Wrap(err, "生成刷新令牌失败")
	}

	// 持久化访问令牌和刷新令牌
	if err := t.persistTokenPair(ctx, &accessClaims, &refreshClaims); err != nil {
		t.logger.Warningf(ctx, "持久化令牌对失败: %v", err)
		// 即使持久化失败，也返回令牌，但记录警告
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// 解析token（仅用于访问令牌）
func (t *Token) Parse(r *ghttp.Request) (*CustomClaims, bool, error) {
	// 从请求中获取token
	token := t.GetTokenFromRequest(r)
	if token == "" {
		t.logger.Debugf(r.GetCtx(), "Token为空")
		return nil, false, gerror.New("Token为空")
	}

	// 解析token
	customClaims, err := t.signer.ParseToken(token)
	if err != nil {
		t.logger.Debugf(r.GetCtx(), err.Error())
		return nil, false, err
	}

	if t.store == nil {
		// 如果没有store，只验证JWT本身的有效性
		return customClaims, true, nil
	}

	record, err := t.store.GetActive(r.GetCtx(), customClaims.ID)
	if err != nil {
		return nil, false, err
	}
	if record == nil {
		return nil, false, nil
	}

	// 确保是访问令牌类型
	if record.TokenType != "" && record.TokenType != "access" {
		return nil, false, gerror.New("无效的令牌类型，此接口仅接受访问令牌")
	}

	tokenInfo := ConvertTokenEntityToModel(record)
	if !tokenInfo.IsActive {
		return nil, false, nil
	}

	return customClaims, true, nil
}

// RevokeToken 按 token 字符串主动撤销
func (t *Token) RevokeToken(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	customClaims, err := t.signer.ParseToken(token)
	if err != nil {
		return err
	}
	if err := t.store.RevokeByJTI(ctx, customClaims.ID); err != nil {
		return err
	}
	return nil
}

// RevokeDeviceToken 强制设备登出
func (t *Token) RevokeDeviceToken(ctx context.Context, deviceID string) error {
	if err := t.store.RevokeByDeviceID(ctx, deviceID); err != nil {
		return err
	}
	return nil
}

// RevokeUserToken 禁用用户
func (t *Token) RevokeUserToken(ctx context.Context, userID string) error {
	if userID == "" {
		return nil
	}
	if t.store != nil {
		if err := t.store.DeactivateByUser(ctx, userID); err != nil {
			return err
		}
	}
	return nil
}

// Refresh 使用刷新令牌生成新的访问令牌和刷新令牌
func (t *Token) Refresh(ctx context.Context, refreshToken string) (pair *TokenPair, err error) {
	if refreshToken == "" {
		return nil, gerror.New("刷新令牌为空")
	}

	// 解析刷新令牌
	refreshClaims, err := t.signer.ParseToken(refreshToken)
	if err != nil {
		t.logger.Debugf(ctx, "解析刷新令牌失败: %v", err)
		return nil, gerror.Wrap(err, "无效的刷新令牌")
	}

	// 验证刷新令牌是否在数据库中且有效
	if t.store == nil {
		return nil, gerror.New("TokenStore未初始化")
	}

	refreshTokenRecord, err := t.store.GetActiveRefreshToken(ctx, refreshClaims.ID)
	if err != nil {
		t.logger.Debugf(ctx, "查询刷新令牌失败: %v", err)
		return nil, gerror.Wrap(err, "查询刷新令牌失败")
	}

	if refreshTokenRecord == nil {
		return nil, gerror.New("刷新令牌不存在或已失效")
	}

	// 检查刷新令牌是否过期
	if time.Now().After(refreshClaims.ExpiresAt.Time) {
		return nil, gerror.New("刷新令牌已过期")
	}

	// 使旧的刷新令牌和关联的访问令牌失效（刷新令牌只能使用一次）
	// 刷新令牌的 refresh_token_id 字段存储的是关联的访问令牌ID
	if refreshTokenRecord.RefreshTokenID != "" {
		if err := t.store.RevokeByJTI(ctx, refreshTokenRecord.RefreshTokenID); err != nil {
			t.logger.Warningf(ctx, "撤销旧访问令牌失败: %v", err)
		}
	}

	// 撤销旧的刷新令牌
	if err := t.store.RevokeByJTI(ctx, refreshClaims.ID); err != nil {
		t.logger.Warningf(ctx, "撤销旧刷新令牌失败: %v", err)
	}

	// 使用旧的自定义数据生成新的令牌对
	return t.Generate(ctx, refreshClaims.Data)
}

// persistTokenPair 持久化令牌对（访问令牌和刷新令牌）
func (t *Token) persistTokenPair(ctx context.Context, accessClaims, refreshClaims *CustomClaims) error {
	if t.store == nil {
		return nil
	}

	userID := accessClaims.Data.(map[string]interface{})["user_id"].(string)
	deviceID := accessClaims.Data.(map[string]interface{})["device_id"].(string)
	accessJTI := accessClaims.ID
	refreshJTI := refreshClaims.ID

	contentBytes, _ := json.Marshal(accessClaims.Data)
	accessExpire := accessClaims.ExpiresAt.Time
	refreshExpire := refreshClaims.ExpiresAt.Time

	// 创建访问令牌记录，关联刷新令牌ID
	if err := t.store.CreateWithType(ctx, accessJTI, userID, deviceID, string(contentBytes), "access", refreshJTI, accessExpire.Unix()); err != nil {
		return err
	}

	// 创建刷新令牌记录，关联访问令牌ID
	if err := t.store.CreateWithType(ctx, refreshJTI, userID, deviceID, string(contentBytes), "refresh", accessJTI, refreshExpire.Unix()); err != nil {
		return err
	}

	return nil
}
