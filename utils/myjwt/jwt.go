package myjwt

import (
	"GopherAI/config"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT
func GenerateToken(id int64, username string) (string, error) {
	claims := Claims{
		ID:       id,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			//过期时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.GetConfig().ExpireDuration) * time.Hour)),
			// 签发人
			Issuer: config.GetConfig().Issuer,
			//主体
			Subject: config.GetConfig().Subject,
			// 签发时间
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}

	// 生成 token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.GetConfig().Key))
}

// ParseToken 解析Token
func ParseToken(token string) (string, bool) {
	claims := new(Claims)
	//ParseWithClaims方法用于解析JWT字符串并将其存储在claims中。它接受三个参数：要解析的JWT字符串、一个指向Claims结构体的指针以及一个回调函数，该函数返回用于验证签名的密钥。
	t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.GetConfig().Key), nil
	})
	//！Vaild表示token过期了
	if !t.Valid || err != nil || claims == nil {
		return "", false
	}
	return claims.Username, true
}
