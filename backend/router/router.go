package router

import (
	"echo/handler"
	"echo/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter 初始化路由，并将各个路由处理函数与对应的中间件绑定
func SetupRouter(h *handler.Handlers) *gin.Engine {
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
			auth.POST("/register", h.User.Register)
			auth.POST("/login", h.User.Login)
		}

		// 除 auth 和 ping 外全部需登录，中间件挂组上
		authed := api.Group("", middleware.JWTAuth())
		{
			// /api/users/me 资源组：当前用户资料与头像（对齐接口文档 §3.4 与路径风格）
			usersMe := authed.Group("/users/me")
			{
				usersMe.GET("", h.Me.Profile)               //当前用户完整资料
				usersMe.PATCH("/avatar", h.Me.UpdateAvatar) //换头像
			}

			// /api/questions 资源组：发布问题（详情、争议列表后续继续挂这
			questions := authed.Group("/questions")
			{
				questions.POST("", h.Question.Create) //发布问题与选项
			}
		}
	}

	return r
}
