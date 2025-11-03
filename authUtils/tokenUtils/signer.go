package tokenUtils

import (
	"context"
	"errors"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/golang-jwt/jwt/v5"
)

// 使用工厂创建一个 JWT 结构体
func CreateMyJWT(JwtTokenSignKey string) *JwtSign {
	return &JwtSign{
		[]byte(JwtTokenSignKey),
	}
}

type JwtSign struct {
	signingKey []byte
}

func (j *JwtSign) CreateToken(claims CustomClaims) (string, error) {
	tokenPartA := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tokenPartA.SignedString(j.signingKey)
}

func (j *JwtSign) ParseToken(tokenString string) (*CustomClaims, error) {
	g.Log().Info(context.Background(), "tokenString", tokenString)
	claimsData, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.signingKey, nil
	})
	if err != nil {
		g.Log().Error(context.Background(), err)
		return nil, err
	}
	if claimsData == nil {
		g.Log().Error(context.Background(), errors.New("claimsData is nil"))
		return nil, errors.New(ErrorsTokenInvalid)
	}
	if claims, ok := claimsData.Claims.(*CustomClaims); ok && claimsData.Valid {
		return claims, nil
	} else {
		g.Log().Error(context.Background(), errors.New("claimsData is not valid"))
		return nil, errors.New(ErrorsTokenInvalid)
	}
}

func (j *JwtSign) RefreshToken(tokenString string, accessTokenTimeout time.Duration) (string, error) {
	customClaims, err := j.ParseToken(tokenString)
	if err != nil {
		return "", err
	}

	customClaims.IssuedAt = jwt.NewNumericDate(time.Now())
	customClaims.NotBefore = jwt.NewNumericDate(time.Now())
	customClaims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(accessTokenTimeout))
	return j.CreateToken(*customClaims)
}
