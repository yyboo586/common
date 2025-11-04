package tokenUtils

import (
	"time"

	"github.com/gogf/gf/v2/errors/gerror"
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
	claimsData, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.signingKey, nil
	})
	if err != nil {
		return nil, gerror.Wrap(err, ErrTokenInvalidText)
	}
	if claimsData == nil {
		return nil, gerror.New(ErrTokenInvalidText + " claimsData is nil")
	}

	if claims, ok := claimsData.Claims.(*CustomClaims); ok && claimsData.Valid {
		return claims, nil
	} else {
		return nil, gerror.New(ErrTokenInvalidText + " claimsData is not CustomClaims type")
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
