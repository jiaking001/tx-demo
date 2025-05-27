package pkg

import (
	"errors"
	"fmt"
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

// ParseJWT 解析并验证JWT令牌
func ParseJWT(tokenString string, j JWT) (string, error) {
	// 解析并验证令牌
	token, err := jwt.ParseWithClaims(tokenString, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.JwtKey, nil
	})

	if err != nil {
		return "", err
	}

	// 类型断言获取声明
	if claims, ok := token.Claims.(*jwt.StandardClaims); ok && token.Valid {
		return claims.Subject, nil // 返回用户ID
	}

	return "", errors.New("invalid token")
}
