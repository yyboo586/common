package jwtUtils

// 测试不同算法，签名的性能

import (
	"crypto/x509"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/yyboo586/common/dbUtils"
)

func TestSignAndVerify(t *testing.T) {
	config := &dbUtils.Config{
		DBName: "test_db",
		Host:   "127.0.0.1",
		Passwd: "12345678",
		Port:   3306,
		User:   "root",
	}
	dbPool, err := dbUtils.NewDB(config)
	if err != nil {
		panic(err)
	}

	SetDBPool(dbPool)

	logicsJWT := NewLogicsJWT()

	claims := make(map[string]interface{})
	claims["user_id"] = "12345678"
	claims["user_name"] = "zhangsan"
	for setID, alg := range map[string]string{"sid1": "RS256", "sid2": "ES256", "sid3": "HS256"} {
		jwtTokenStr, err := logicsJWT.Sign("12345678", claims, setID, alg)
		if err != nil {
			panic(err)
		}
		c, err := logicsJWT.Verify(jwtTokenStr)
		if err != nil {
			panic(err)
		}

		assert.Equal(t, c["user_id"], "12345678")
		assert.Equal(t, c["user_name"], "zhangsan")
	}
}

func signForTest(userID string, claims map[string]interface{}, key *jose.JSONWebKey) (jwtTokenStr string, err error) {
	signer, err := jose.NewSigner(jose.SigningKey{Key: key.Key, Algorithm: jose.SignatureAlgorithm(key.Algorithm)}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", key.KeyID))
	if err != nil {
		return
	}

	if claims == nil {
		claims = make(map[string]interface{})
	}

	cclaims := CustomClaims{
		jwt.Claims{
			Issuer:    "example.com",
			Subject:   userID,
			Audience:  []string{"UserManagement"},
			Expiry:    jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		claims,
	}
	jwtTokenStr, err = jwt.Signed(signer).Claims(cclaims).Serialize()
	if err != nil {
		return
	}

	return jwtTokenStr, nil
}

func generateJWKSetForTest(setID, alg, kid, use string) (kSet *jose.JSONWebKeySet, err error) {
	if len(kid) == 0 {
		kid = uuid.Must(uuid.NewV4()).String()
	}
	if len(use) == 0 {
		use = "sig"
	}

	var key interface{}
	switch jose.SignatureAlgorithm(alg) {
	case jose.HS256, jose.HS384, jose.HS512:
		if key, err = NewSymmetricKey(jose.SignatureAlgorithm(alg)); err != nil {
			return nil, err
		}
	default:
		if _, key, err = NewAsymmetricKey(jose.SignatureAlgorithm(alg)); err != nil {
			return nil, err
		}
	}

	kSet = &jose.JSONWebKeySet{
		Keys: []jose.JSONWebKey{
			{
				Algorithm:                   string(alg),
				Key:                         key,
				KeyID:                       kid,
				Use:                         use,
				Certificates:                []*x509.Certificate{},
				CertificateThumbprintSHA256: []byte{},
				CertificateThumbprintSHA1:   []byte{},
			},
		},
	}

	return kSet, nil
}

var (
	userID = "12347"
	claims = map[string]interface{}{
		"username": "john_doe",
		"email":    "john@example.com",
	}
)

func BenchmarkSign(b *testing.B) {
	tests := []struct {
		name string
		alg  string
	}{
		{"HS256", "HS256"},
		{"HS384", "HS384"},
		{"HS512", "HS512"},
		{"ES256", "ES256"},
		{"ES384", "ES384"},
		{"ES512", "ES512"},
		{"RS256", "RS256"},
		{"RS384", "RS384"},
		{"RS512", "RS512"},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			key, err := generateJWKSetForTest("", tt.alg, "", "")
			if err != nil {
				b.Fatal(err)
			}
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := signForTest(userID, claims, &key.Keys[0])
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
