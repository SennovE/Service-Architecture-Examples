package utils

import (
	"crypto/rsa"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrJWTDecode = errors.New("invalid jwt")
var ErrJWTExpired = errors.New("token expired")

func parseToken(pub *rsa.PublicKey, tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, ErrJWTDecode
		}
		return pub, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrJWTExpired
		}
		return nil, ErrJWTDecode
	}

	if !token.Valid {
		return nil, ErrJWTDecode
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrJWTDecode
	}

	return claims, nil
}

func VerifyAccessToken(pub *rsa.PublicKey, tokenStr string) (jwt.MapClaims, error) {
	claims, err := parseToken(pub, tokenStr)
	if err != nil {
		return nil, err
	}

	if claims["typ"] != "access" {
		return nil, ErrJWTDecode
	}

	return claims, nil
}

func VerifyRefreshToken(pub *rsa.PublicKey, tokenStr string) (jwt.MapClaims, error) {
	claims, err := parseToken(pub, tokenStr)
	if err != nil {
		return nil, err
	}

	if claims["typ"] != "refresh" {
		return nil, ErrJWTDecode
	}

	return claims, nil
}

func CreateAccessToken(priv *rsa.PrivateKey, accessTTL time.Duration, userID uuid.UUID, role string) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"typ":  "access",
		"iat":  now.Unix(),
		"exp":  now.Add(accessTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(priv)
}

func CreateRefreshToken(priv *rsa.PrivateKey, refreshTTL time.Duration, userID uuid.UUID) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"sub": userID,
		"typ": "refresh",
		"iat": now.Unix(),
		"exp": now.Add(refreshTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(priv)
}
