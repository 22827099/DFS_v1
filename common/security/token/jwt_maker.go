package token

import (
	"time"
	"errors"

	"github.com/golang-jwt/jwt/v4"
)

// JWTMaker 是使用JWT算法的令牌制造商
type JWTMaker struct {
	secretKey string
	issuer    string
}

// NewJWTMaker 创建一个新的JWTMaker
func NewJWTMaker(secretKey string) (Maker, error) {
	if len(secretKey) < 32 {
		// 生产环境应该使用至少32字节的密钥
		return nil, errors.New("密钥长度不足，至少需要32字节")
	}

	return &JWTMaker{secretKey: secretKey}, nil
}

// NewJWTMakerWithIssuer 创建一个带有颁发者的JWTMaker
func NewJWTMakerWithIssuer(secretKey string, issuer string) (Maker, error) {
	if len(secretKey) < 32 {
		return nil, errors.New("密钥长度不足，至少需要32字节")
	}

	return &JWTMaker{
		secretKey: secretKey,
		issuer:    issuer,
	}, nil
}

// CreateToken 实现Maker接口
func (maker *JWTMaker) CreateToken(username, subject string, duration time.Duration) (string, error) {
	payload, err := NewPayload(username, subject, duration, maker.issuer)
	if err != nil {
		return "", err
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return jwtToken.SignedString([]byte(maker.secretKey))
}

// VerifyToken 实现Maker接口
func (maker *JWTMaker) VerifyToken(token string) (*Payload, error) {
	keyFunc := func(token *jwt.Token) (interface{}, error) {
		_, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok {
			return nil, ErrInvalidSigningMethod
		}
		return []byte(maker.secretKey), nil
	}

	jwtToken, err := jwt.ParseWithClaims(token, &Payload{}, keyFunc)
	if err != nil {
		// 使用类型断言检查是否为ValidationError
		validationErr, ok := err.(*jwt.ValidationError)
		if ok && validationErr.Errors&jwt.ValidationErrorExpired != 0 {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	payload, ok := jwtToken.Claims.(*Payload)
	if !ok {
		return nil, ErrInvalidToken
	}

	return payload, nil
}
