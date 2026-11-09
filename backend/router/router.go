package router

import (
	"echo/handler"
	"echo/middleware"

	"github.com/gin-gonic/gin"
)

func SetupRouter(userHandler *handler.UserHandler) *gin.Engine {
	r := gin.New()

	//挂载基建
	r.Use(middleware.Cors())
	r.Use(middleware.Logger())

	//免登录的区域
	public := r.Group("/api/public")
	{
		public.GET("/ping", func(c *gin.Context) {
			handler.Success(c, "pong 后端基建运行正常")
		})
		public.POST("/register", userHandler.Register)
	}

	//需鉴权的区域
	private := r.Group("/api/private")

	_ = private //这行用来测试的，我不声明的话go会报错，这个暂时不需要，不用管

	// private.Use(middleware.JWTAuth()) //等要打开的时候我再打开
	{
	}

	return r
}
