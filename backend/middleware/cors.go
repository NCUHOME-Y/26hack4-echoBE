package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

//Cors 跨域中间件。允许的源由 CORS_ALLOW_ORIGIN 注入，逗号分隔；留空表示放开。
func Cors() gin.HandlerFunc {
	//环境变量在 SetupRouter 之前就已经加载（repository.NewDB 里那次 godotenv.Load），
	//所以这里解析一次就够，不必每个请求再切一遍字符串
	allowed := parseAllowedOrigins(os.Getenv("CORS_ALLOW_ORIGIN"))

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		switch {
		case allowed == nil:
			//未配置就是放开，行为与加白名单之前完全一致
			c.Header("Access-Control-Allow-Origin", "*")
		case origin != "" && containsOrigin(allowed, origin):
			//命中白名单才把对方的源原样回写。这里不能回 *，也不能回一个不在名单里的源，
			//否则等于让浏览器替你放行
			c.Header("Access-Control-Allow-Origin", origin)
			//响应随请求头里的 Origin 变化，必须声明 Vary，否则 CDN 或浏览器缓存
			//会把给 A 站的放行结果发给 B 站
			c.Header("Vary", "Origin")
		default:
			//不带 Origin 的请求（curl、服务端调用、同源页面）本来就不需要 CORS 头；
			//跨站但不在名单里的请求什么都不设，让浏览器拦掉，接口自身照常返回
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if reqHeaders := c.GetHeader("Access-Control-Request-Headers"); reqHeaders != "" {
			//预检会把它要用的头报过来，照着回写最准；否则新增一个自定义头就得改代码
			c.Header("Access-Control-Allow-Headers", reqHeaders)
		} else {
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

//parseAllowedOrigins 把逗号分隔的来源列表切成小写数组
func parseAllowedOrigins(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	list := make([]string, 0, 4)
	for _, item := range strings.Split(value, ",") {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if item == "*" {
			return nil
		}
		list = append(list, item)
	}
	if len(list) == 0 {
		return nil
	}
	return list
}

//containsOrigin 精确比对来源。这里不做通配符匹配：
//带子域通配要引一套模式匹配规则，写错一个点就变成"顺手放开了整个域"。
func containsOrigin(allowed []string, origin string) bool {
	target := strings.ToLower(strings.TrimSpace(origin))
	for _, a := range allowed {
		if a == target {
			return true
		}
	}
	return false
}
