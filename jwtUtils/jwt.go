package jwtUtils

import (
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

type CustomClaims struct {
	jwt.Claims

	ExtClaims map[string]interface{}
}

var dbPool *sql.DB

func SetDBPool(pool *sql.DB) {
	dbPool = pool
}

type LogicsJWT interface {
	Sign(userID string, claims map[string]interface{}, setID, alg string) (jwtTokenStr string, err error)
	Verify(jwtTokenStr string) (extClaims map[string]interface{}, err error)
}

var (
	logicsJWTOnce sync.Once
	lJWT          LogicsJWT
)

type logicsJWT struct {
	dbJWT     DBJWT
	keysCache map[string]*jose.JSONWebKeySet
}

func NewLogicsJWT() LogicsJWT {
	logicsJWTOnce.Do(func() {
		lJWT = &logicsJWT{
			dbJWT:     NewDBJWT(),
			keysCache: make(map[string]*jose.JSONWebKeySet),
		}
	})

	return lJWT
}

func (j *logicsJWT) Sign(userID string, claims map[string]interface{}, setID, alg string) (jwtTokenStr string, err error) {
	key, err := j.loadORGenerateKeys(setID, alg)
	if err != nil {
		return "", err
	}

	signer, err := jose.NewSigner(jose.SigningKey{Key: key.Key, Algorithm: jose.SignatureAlgorithm(alg)}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", key.KeyID))
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

func getKid(jwtTokenStr string) (alg string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("GetKid error: %v", r)
		}
	}()

	headerBase64 := strings.Split(jwtTokenStr, ".")[0]

	headerBytes, err := base64.RawURLEncoding.DecodeString(headerBase64)
	if err != nil {
		return "", fmt.Errorf("decode header failed: %w", err)
	}

	var header map[string]interface{}
	if err = json.Unmarshal(headerBytes, &header); err != nil {
		return "", fmt.Errorf("unmarshal header failed: %w", err)
	}

	return header["kid"].(string), nil
}

func (j *logicsJWT) getKey(kid string) (key *jose.JSONWebKey, err error) {
	for _, v := range j.keysCache {
		for _, key := range v.Keys {
			if key.KeyID == kid {
				return &key, nil
			}
		}
	}

	key, err = j.dbJWT.GetKey(kid)
	if err != nil {
		return nil, err
	}

	return key, nil
}

func (j *logicsJWT) Verify(jwtTokenStr string) (extClaims map[string]interface{}, err error) {
	kid, err := getKid(jwtTokenStr)
	if err != nil {
		return
	}
	key, err := j.getKey(kid)
	if err != nil {
		return nil, err
	}

	jwtToken, err := jwt.ParseSigned(jwtTokenStr, []jose.SignatureAlgorithm{jose.SignatureAlgorithm(key.Algorithm)})
	if err != nil {
		return
	}

	var cclaims CustomClaims
	switch key.Algorithm {
	case "HS256", "HS384", "HS512":
		err = jwtToken.Claims(key.Key, &cclaims)
	default:
		err = jwtToken.Claims(key.Public(), &cclaims)
	}
	if err != nil {
		return
	}
	expected := jwt.Expected{
		Issuer:      "example.com",
		AnyAudience: jwt.Audience{"UserManagement"},
		Time:        time.Time{},
	}
	if err = cclaims.Claims.Validate(expected); err != nil {
		return
	}

	return cclaims.ExtClaims, err
}

// logics layer
func (j *logicsJWT) loadORGenerateKeys(setID, alg string) (key *jose.JSONWebKey, err error) {
	if kSet, ok := j.keysCache[setID]; ok {
		return &kSet.Keys[0], nil
	}

	kSet, err := j.dbJWT.GetKeySet(setID)
	if err != nil {
		return nil, err
	}
	if len(kSet.Keys) == 0 {
		kid := uuid.Must(uuid.NewV4()).String()
		kSet, err = j.generateAndPersistJWKSet(setID, alg, kid, "sig")
		if err != nil {
			return nil, err
		}
	}

	j.keysCache[setID] = kSet

	return &kSet.Keys[0], nil
}

// logics layer
func (j *logicsJWT) generateAndPersistJWKSet(setID, alg, kid, use string) (kSet *jose.JSONWebKeySet, err error) {
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

	if err = j.dbJWT.AddKeySet(setID, kSet); err != nil {
		return nil, err
	}
	return kSet, nil
}

type DBJWT interface {
	AddKeySet(setID string, keySet *jose.JSONWebKeySet) error
	GetKeySet(setID string) (kSet *jose.JSONWebKeySet, err error)
	GetKey(kid string) (key *jose.JSONWebKey, err error)
}

var (
	dbJWTOnce sync.Once
	dJWT      *dbJWT
)

type dbJWT struct {
	dbPool *sql.DB
}

func NewDBJWT() DBJWT {
	dbJWTOnce.Do(func() {
		dJWT = &dbJWT{
			dbPool: dbPool,
		}
	})
	return dJWT
}

// todo: transaction
func (j *dbJWT) AddKeySet(setID string, keySet *jose.JSONWebKeySet) (err error) {
	if len(keySet.Keys) == 0 {
		return nil
	}

	format := []string{}
	values := []any{}
	for _, key := range keySet.Keys {
		data, err := json.Marshal(key)
		if err != nil {
			return err
		}

		format = append(format, "(?, ?, ?)")
		values = append(values, key.KeyID, string(data), setID)
	}
	sqlStr := "INSERT INTO t_jwt_keys(id, data, sid) VALUES" + strings.Join(format, ",")

	if _, err = j.dbPool.Exec(sqlStr, values...); err != nil {
		return err
	}

	return nil
}

func (j *dbJWT) GetKeySet(setID string) (kSet *jose.JSONWebKeySet, err error) {
	sqlStr := "SELECT data FROM t_jwt_keys WHERE sid = ? ORDER BY created_at DESC"

	rows, err := j.dbPool.Query(sqlStr, setID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	kSet = &jose.JSONWebKeySet{}
	for rows.Next() {
		var data string
		if err = rows.Scan(&data); err != nil {
			return nil, err
		}

		key := jose.JSONWebKey{}
		if err = json.Unmarshal([]byte(data), &key); err != nil {
			return nil, err
		}

		kSet.Keys = append(kSet.Keys, key)
	}

	return kSet, nil
}

func (j *dbJWT) GetKey(kid string) (key *jose.JSONWebKey, err error) {
	sqlStr := "SELECT data FROM t_jwt_keys WHERE id = ?"

	var data string
	if err = j.dbPool.QueryRow(sqlStr, kid).Scan(&data); err != nil {
		return nil, err
	}

	key = &jose.JSONWebKey{}
	if err = json.Unmarshal([]byte(data), &key); err != nil {
		return nil, err
	}

	return key, nil
}

func Sign(userID string, claims map[string]interface{}, privateKey *rsa.PrivateKey) (jwtTokenStr string, err error) {
	signer, err := jose.NewSigner(jose.SigningKey{Key: privateKey, Algorithm: jose.RS256}, (&jose.SignerOptions{}).WithType("JWT"))
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
			Audience:  []string{"IAMService"},
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

func Verify(jwtTokenStr string, privateKey *rsa.PrivateKey) (extClaims map[string]interface{}, err error) {
	jwtToken, err := jwt.ParseSigned(jwtTokenStr, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		return
	}

	var cclaims CustomClaims
	if err = jwtToken.Claims(&privateKey.PublicKey, &cclaims); err != nil {
		return
	}
	expected := jwt.Expected{
		Issuer:      "example.com",
		AnyAudience: jwt.Audience{"IAMService"},
		Time:        time.Time{},
	}
	if err = cclaims.Claims.Validate(expected); err != nil {
		return
	}

	return cclaims.ExtClaims, err
}
