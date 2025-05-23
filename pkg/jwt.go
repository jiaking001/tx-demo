package pkg

import (
	"github.com/dgrijalva/jwt-go"
	"time"
)

type JWT struct {
	JwtIssuer string
	JwtKey    []byte
}

// GenerateJWT 生成JWT令牌
func GenerateJWT(userID string, j JWT) (string, int64, error) {
	expiresAt := time.Now().Add(time.Hour * 24).Unix()

	claims := &jwt.StandardClaims{
		Subject:   userID,
		ExpiresAt: expiresAt,
		Issuer:    j.JwtIssuer,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(j.JwtKey)
	if err != nil {
		return "", 0, err
	}

	return tokenString, expiresAt - time.Now().Unix(), nil
}
