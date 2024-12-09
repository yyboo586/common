package jwtUtils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"math/big"

	"github.com/go-jose/go-jose/v4"
)

func NewAsymmetricKey(alg jose.SignatureAlgorithm) (crypto.PublicKey, crypto.PrivateKey, error) {
	switch alg {
	case jose.ES256:
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		return key.Public(), key, nil
	case jose.ES384:
		key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		return key.Public(), key, nil
	case jose.ES512:
		key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		if err != nil {
			return nil, nil, err
		}
		return key.Public(), key, nil
	case jose.RS256, jose.RS384, jose.RS512:
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, nil, err
		}
		return key.Public(), key, nil
	default:
		return nil, nil, fmt.Errorf("unsupported algorithm %s for asymmetric key", alg)
	}
}

// - HS256: 32 bytes
// - HS384: 48 bytes
// - HS512: 64 bytes
func NewSymmetricKey(alg jose.SignatureAlgorithm) ([]byte, error) {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	var length int
	switch alg {
	case jose.HS256:
		length = 32
	case jose.HS384:
		length = 48
	case jose.HS512:
		length = 64
	default:
		return nil, fmt.Errorf("unsupported algorithm %s for symmetric key", alg)
	}

	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return nil, err
		}
		b[i] = letterBytes[num.Int64()]
	}
	return b, nil
}
