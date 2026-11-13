package main

import (
	"log"
	"os"

	"echo/handler"
	"echo/middleware"
	"echo/repository"
	"echo/router"
	"echo/service"

	"go.uber.org/zap"
)

func main() {

	zapLogger, err := zap.NewProduction() //创建zap（生产格式：json
	if err != nil {
		log.Fatalf("zap日志初始化失败: %v", err)
	}
	defer zapLogger.Sync() //程序退出前把缓冲区里的日志全部刷出去 防丢

	middleware.InitLogger(zapLogger)

	db, err := repository.NewDB() //建立数据库连接
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	//godotenv 是在 NewDB 里加载的，所以这个检查必须放在它后面
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("缺少 JWT_SECRET，请检查 .env")
	}

	userRepo := repository.NewUserRepository(db)   //创建用户仓库
	userSvc := service.NewUserService(userRepo)    //创建用户服务
	userHandler := handler.NewUserHandler(userSvc) //创建用户处理器（这里三个是单向向下的依赖关系）
	meSvc := service.NewMeService(userRepo)        //创建我的服务
	meHandler := handler.NewMeHandler(meSvc)

	//创建四个投票模块的数据库连接
	questionRepo := repository.NewQuestionRepository(db)
	optionRepo := repository.NewOptionRepository(db)
	voteRepo := repository.NewVoteRepository(db)
	skipRepo := repository.NewSkipRepository(db)
	//创建投票模块的服务
	voteSvc := service.NewVoteService(questionRepo, optionRepo, voteRepo, skipRepo)
	//创建投票模块的handler
	voteHandler := handler.NewVoteHandler(voteSvc)

	//handler 全部收进一个结构体交给路由，以后每加一个模块只改这一处，不用再去动 SetupRouter 的签名
	handlers := &handler.Handlers{User: userHandler, Me: meHandler, Vote: voteHandler}
	questionRepo := repository.NewQuestionRepository(db)
	questionSvc := service.NewQuestionService(questionRepo, userRepo)
	questionHandler := handler.NewQuestionHandler(questionSvc)

	//handler 全部收进一个结构体交给路由，以后每加一个模块只改这一处，不用再去动 SetupRouter 的签名
	handlers := &handler.Handlers{User: userHandler, Me: meHandler, Question: questionHandler}

	r := router.SetupRouter(handlers)
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("HTTP 服务启动失败: %v", err)
	}
}
