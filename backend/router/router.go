package router

import (
	"echo/handler"
	"echo/middleware"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRouter 初始化路由，并将各个路由处理函数与对应的中间件绑定
func SetupRouter(h *handler.Handlers) *gin.Engine {
	r := gin.New()
	//panic 兜底：任何一处空指针/越界都不会让进程崩掉，统一返回 JSON 500
	r.Use(gin.CustomRecovery(func(c *gin.Context, err any) {
		log.Printf("panic 已恢复：%s %s err=%v", c.Request.Method, c.Request.URL.Path, err)
		handler.Error(c, http.StatusInternalServerError, "服务端内部错误")
	}))
	r.Use(middleware.Cors())
	r.Use(middleware.Logger())

	r.HandleMethodNotAllowed = true //不开这个开关，NoMethod 永远不会触发

	//路径写错时也返回统一的 {code,msg,data}
	r.NoRoute(func(c *gin.Context) {
		handler.Error(c, http.StatusNotFound, "接口不存在")
	})
	r.NoMethod(func(c *gin.Context) {
		handler.Error(c, http.StatusMethodNotAllowed, "请求方法不允许")
	})

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
				usersMe.GET("", h.Me.Profile)                   //当前用户完整资料
				usersMe.PATCH("/avatar", h.Me.UpdateAvatar)     //换头像
				usersMe.GET("/questions", h.Me.ListMyQuestions) //获取我发布的问题列表
				usersMe.GET("/answers", h.Me.ListMyAnswers)     //我投过票的问题历史
			}

			// /api/questions 资源组：发布、抽题、详情、投票、跳过、评论列表与发表、公布最终选择、争议列表（删除评论后续挂这里）
			questions := authed.Group("/questions")
			{
				questions.POST("", h.Question.Create)             //发布问题与选项
				questions.GET("/random", h.Question.Random)       //随机抽一题，静态路径优先于下面的 :id
				questions.GET("/contested", h.Question.Contested) //争议海螺列表，最新 100 条
				questions.GET("/:id", h.Question.Detail)          //问题详情与实时占比
				questions.POST("/:id/votes", h.Vote.Create)       //提交投票
				questions.POST("/:id/skips", h.Skip.Create)       //跳过这题，永久放弃作答权
				questions.GET("/:id/replies", h.Reply.List)       //评论列表，最多返回最新 50 条
				questions.POST("/:id/replies", h.Reply.Create)    //发表回复
				questions.PUT("/:id/final", h.Final.Publish)      //公布最终选择，公布即结束

			}

			// /api/replies 资源组：删除回复（评论列表与发表挂在 /questions/:id/replies 下）
			replies := authed.Group("/replies")
			{
				replies.DELETE("/:id", h.Reply.Delete) //软删除一条回复，作者或本题提问者可删

			}

			// /api/stats 资源组：首页看板的今日统计
			stats := authed.Group("/stats")
			{
				stats.GET("/daily", h.Stats.Daily) //今日发布量、投票量、结算量
			}
		}
	}

	return r
}
