package tokenUtils

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gogf/gf/v2/crypto/gaes"
	"github.com/gogf/gf/v2/crypto/gmd5"
	"github.com/gogf/gf/v2/encoding/gbase64"
	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/grand"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/yyboo586/common/cacheUtils"
)

type Token struct {
	cachePrefix          string            // 缓存前缀
	cache                cacheUtils.ICache // 缓存
	excludePaths         g.SliceStr        // 拦截排除地址
	userJwt              *JwtSign          // jwt
	accessTokenLifeSpan  time.Duration     // access token 生命周期
	refreshTokenLifeSpan time.Duration     // refresh token 生命周期
}

// TokenData Token 数据
type tokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// 生成 access token 和 refresh token
func (t *Token) Generate(ctx context.Context, data interface{}) (token string, err error) {
	jti := uuid.New().String()
	token, err = t.userJwt.CreateToken(CustomClaims{
		data,
		jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(time.Unix(time.Now().Unix()-10, 0)),    // 生效开始时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(t.accessTokenLifeSpan)), // 失效截止时间
			ID:        jti,                                                       // jwt令牌唯一标识
		},
	})
	if err != nil {
		return
	}

	t.setCache(ctx, fmt.Sprintf("%s_ac_%s", t.cachePrefix, jti), cacheData, t.accessTokenLifeSpan)
	t.setCache(ctx, fmt.Sprintf("%s_rt_%s", t.cachePrefix, jti), cacheData, t.refreshTokenLifeSpan)

	return
}

// 解析token (只验证格式并不验证过期)
func (m *Token) ParseToken(r *ghttp.Request) (*CustomClaims, error) {
	token, err := m.GetToken(r)
	log.Println(token)
	log.Println(err)
	if err != nil {
		return nil, err
	}
	if customClaims, err := m.userJwt.ParseToken(token.JwtToken); err == nil {
		return customClaims, nil
	} else {
		return &CustomClaims{}, errors.New(ErrorsParseTokenFail)
	}
}

// 检查缓存的token是否有效且自动刷新缓存token
func (m *Token) IsEffective(ctx context.Context, token string) bool {
	cacheToken, key, err := m.GetTokenData(ctx, token)
	if err != nil {
		g.Log().Info(ctx, err)
		return false
	}
	_, code := m.IsNotExpired(cacheToken.JwtToken)
	if JwtTokenOK == code {
		// 刷新缓存
		if m.IsRefresh(cacheToken.JwtToken) {
			return m.doRefresh(ctx, key, cacheToken)
		}
		return true
	}
	return false
}

func (m *Token) doRefresh(ctx context.Context, key string, cacheToken *TokenData) bool {
	if newToken, err := m.RefreshToken(cacheToken.JwtToken); err == nil {
		cacheToken.JwtToken = newToken
		err = m.setCache(ctx, m.CacheKey+key, cacheToken)
		if err != nil {
			g.Log().Error(ctx, err)
			return false
		}
	}
	return true
}

func (m *Token) GetTokenData(ctx context.Context, token string) (tData *TokenData, key string, err error) {
	var uuid string
	key, uuid, err = m.DecryptToken(ctx, token)
	if err != nil {
		return
	}
	tData, err = m.getCache(ctx, m.cachePrefix+key)
	if tData == nil || tData.UuId != uuid {
		err = gerror.New("token is invalid")
	}
	return
}

// 检查token是否过期 (过期时间 = 超时时间 + 缓存刷新时间)
func (m *Token) IsNotExpired(token string) (*CustomClaims, int) {
	if customClaims, err := m.userJwt.ParseToken(token); err == nil {
		if time.Now().Unix()-customClaims.ExpiresAt.Unix() < 0 {
			// token有效
			return customClaims, JwtTokenOK
		} else {
			// 过期的token
			return customClaims, JwtTokenExpired
		}
	} else {
		// 无效的token
		return customClaims, JwtTokenInvalid
	}
}

// 刷新token的缓存有效期
func (m *Token) RefreshToken(oldToken string) (newToken string, err error) {
	if newToken, err = m.userJwt.RefreshToken(oldToken, m.diedLine().Unix()); err != nil {
		return
	}
	return
}

// token是否处于刷新期
func (m *Token) IsRefresh(token string) bool {
	if m.MaxRefresh == 0 {
		return false
	}
	if customClaims, err := m.userJwt.ParseToken(token); err == nil {
		now := time.Now().Unix()
		if now < customClaims.ExpiresAt.Unix() && now > (customClaims.ExpiresAt.Unix()-m.MaxRefresh) {
			return true
		}
	}
	return false
}

// EncryptToken token加密方法
func (m *Token) EncryptToken(ctx context.Context, key string, randStr ...string) (encryptStr, uuid string, err error) {
	if key == "" {
		err = gerror.New("encrypt key empty")
		return
	}
	// 生成随机串
	if len(randStr) > 0 {
		uuid = randStr[0]
	} else {
		uuid = gmd5.MustEncrypt(grand.Letters(10))
	}
	token, err := gaes.Encrypt([]byte(key+uuid), m.EncryptKey)
	if err != nil {
		g.Log().Error(ctx, "[GFToken]encrypt error Token:", key, err)
		err = gerror.New("encrypt error")
		return
	}
	encryptStr = gbase64.EncodeToString(token)
	return
}

// DecryptToken token解密方法
func (m *Token) DecryptToken(ctx context.Context, token string) (DecryptStr, uuid string, err error) {
	if token == "" {
		err = gerror.New("decrypt Token empty")
		return
	}
	token64, err := gbase64.Decode([]byte(token))
	if err != nil {
		g.Log().Info(ctx, "[GFToken]decode error Token:", token, err)
		err = gerror.New("decode error")
		return
	}
	decryptToken, err := gaes.Decrypt(token64, m.EncryptKey)
	if err != nil {
		g.Log().Info(ctx, "[GFToken]decrypt error Token:", token, err)
		err = gerror.New("decrypt error")
		return
	}
	length := len(decryptToken)
	uuid = string(decryptToken[length-32:])
	DecryptStr = string(decryptToken[:length-32])
	return
}

// RemoveToken 删除token
func (m *Token) RemoveToken(ctx context.Context, token string) (err error) {
	var key string
	_, key, err = m.GetTokenData(ctx, token)
	if err != nil {
		return
	}
	err = m.removeCache(ctx, m.CacheKey+key)
	return
}
