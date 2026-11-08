package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		c.Next()
		log.Printf("[Echo API] 路径:%s | 耗时:%v | 状态码:%d", path, time.Since(start), c.Writer.Status())
	}
}
