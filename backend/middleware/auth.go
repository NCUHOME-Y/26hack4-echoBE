package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

//定义JWT所携带的数据
type Claims struct {
	UserID int64 `json:"user_id"`
	jwt.RegisteredClaims //JWT所提供的标准字段集合（本代码暂时只使用了签发与过期时间）
}

//tokenTTL 登录凭证有效期。默认 24 小时
func tokenTTL() time.Duration {
	h, err := strconv.Atoi(os.Getenv("JWT_EXPIRE_HOURS"))
	if err != nil || h <= 0 {
		h = 24
	}
	return time.Duration(h) * time.Hour
}

//生成登录凭证
func GenerateToken(userID int64) (string, error) {
	//先读取环境变量里面的JWT密钥
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return "", fmt.Errorf("缺少 JWT_SECRET，请检查 .env")
	}

	//生成Claims
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()), //签发时间
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL())), //有效期由 JWT_EXPIRE_HOURS 决定，默认 24 小时
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret)) //使用 HS256 签名
}

//JWT鉴权中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		const prefix = "Bearer " //固定前缀
		header := c.GetHeader("Authorization") //读取用户请求头
		if !strings.HasPrefix(header, prefix) { //检查前缀是否正确
			abortUnauthorized(c, "未登录：缺少 Bearer token")
			return
		}

		//解析JWT，将Payload解析到Claims结构体中
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(strings.TrimPrefix(header, prefix), claims, func(t *jwt.Token) (interface{}, error) {
			//检查签名算法是否正确
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("意外的签名算法: %v", t.Header["alg"])
			}
			//密钥为空时验签会用空密钥，攻击者可自行伪造 token，必须拒绝
			secret := os.Getenv("JWT_SECRET")
			if secret == "" {
				return nil, fmt.Errorf("JWT_SECRET 未配置")
			}
			return []byte(secret), nil
		})
		//检查token是否有效
		if err != nil || !token.Valid {
			abortUnauthorized(c, "未登录或凭证已过期")
			return
		}

		c.Set("userID", claims.UserID) //把用户ID存到上下文中，后续的处理函数可以直接使用
		c.Next()
	}
}

//统一返回未授权响应
func abortUnauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"code": http.StatusUnauthorized, "msg": msg, "data": nil,
	})
}
