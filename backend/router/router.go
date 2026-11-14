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
			// /api/users/me 资源组：当前用户资料与头像
			usersMe := authed.Group("/users/me")
			{
				usersMe.GET("", h.Me.Profile)               //当前用户完整资料
				usersMe.PATCH("/avatar", h.Me.UpdateAvatar) //换头像
			}

			// /api/questions 资源组：发布、抽题、详情、投票、跳过、评论列表（争议列表、结算、发表与删除评论后续挂这里）
			questions := authed.Group("/questions")
			{
				questions.POST("", h.Question.Create)          //发布问题与选项
				questions.GET("/random", h.Question.Random)    //随机抽一题，静态路径优先于下面的 :id
				questions.GET("/:id", h.Question.Detail)       //问题详情与实时占比
				questions.POST("/:id/votes", h.Vote.Create)    //提交投票
				questions.POST("/:id/skips", h.Skip.Create)    //跳过这题，永久放弃作答权
				questions.GET("/:id/replies", h.Reply.List)    //评论列表，最多返回最新 50 条
				questions.POST("/:id/replies", h.Reply.Create) //发表回复

			}
		}
	}

	return r
}
