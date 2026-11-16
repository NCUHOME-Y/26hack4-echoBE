package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var zapLogger *zap.Logger //全局日志器 由main启动时初始化注入

func InitLogger(l *zap.Logger) { //在程序启动时调用 zap实例交给middleware包用

	zapLogger = l
}

//L 取全局日志器。没初始化时返回一个什么都不写的 Nop logger
func L() *zap.Logger {
	if zapLogger == nil {
		return zap.NewNop()
	}
	return zapLogger
}

func Logger() gin.HandlerFunc { //记录每个http耗时和状态码
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next() //先放行 等请求处理完成再打日志 这样才能拿状态码和总耗时

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("duration", time.Since(start)),
		}
		if len(c.Errors) > 0 {
			//c.Error 记过的业务错误跟着请求行一起出来，否则光看 status=500 不知道错在哪
			fields = append(fields, zap.Strings("errors", c.Errors.Errors()))
		}
		L().Info("http请求", fields...)
	}
}
