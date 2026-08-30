// package main

// import (
// 	"GopherAI/config"
// 	"fmt"
// 	"time"

// 	"github.com/golang-jwt/jwt/v4"
// )

// type Claims struct {
// 	ID       int64  `json:"id"`
// 	Username string `json:"username"`
// 	jwt.RegisteredClaims
// }

// // GenerateToken 生成JWT
// func GenerateToken(id int64, username string) (string, error) {
// 	claims := Claims{
// 		ID:       id,
// 		Username: username,
// 		RegisteredClaims: jwt.RegisteredClaims{
// 			//过期时间
// 			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.GetConfig().ExpireDuration) * time.Hour)),
// 			// 签发人
// 			Issuer: config.GetConfig().Issuer,
// 			//主体
// 			Subject: config.GetConfig().Subject,
// 			// 签发时间
// 			IssuedAt: jwt.NewNumericDate(time.Now()),
// 		},
// 	}

// 	// 生成 token
// 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

// 	return token.SignedString([]byte(config.GetConfig().Key))

// }

// func main() {
// 	token, _ := GenerateToken(521, "lph")
// 	fmt.Println("token 是", token)
// }

package main

import (
	"fmt"
)

func main() {
	apiURL := fmt.Sprintf(
		"https://wttr.in/%s?format=j1&lang=zh", "beijing",
	)
	fmt.Println(apiURL)
}
