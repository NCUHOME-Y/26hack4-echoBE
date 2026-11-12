package router

import (
	"echo/handler"
	"echo/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter 初始化路由，并将各个路由处理函数与对应的中间件绑定
func SetupRouter(userHandler *handler.UserHandler, meHandler *handler.MeHandler) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Cors())
	r.Use(middleware.Logger())

	api := r.Group("/api")
	{
		api.GET("/ping", func(c *gin.Context) {
			handler.Success(c, "pong 后端基建运行正常")
		})

		auth := api.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
		}

		// 除 auth 和 ping 外全部需登录，中间件挂组上
		authed := api.Group("", middleware.JWTAuth())
		{
			authed.GET("/users/me", userHandler.WhoAmI)

			me := authed.Group("/me")
			{
				me.GET("/profile", meHandler.Profile)       //"我的"页资料
				me.PATCH("/avatar", meHandler.UpdateAvatar) //换头像
			}
		}
	}

	return r
}
