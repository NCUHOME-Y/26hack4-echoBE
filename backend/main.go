package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

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
	meRepo := repository.NewMeRepository(db)
	meSvc := service.NewMeService(userRepo, meRepo)
	meHandler := handler.NewMeHandler(meSvc)

	//创建投票模块的数据库连接
	questionRepo := repository.NewQuestionRepository(db)
	optionRepo := repository.NewOptionRepository(db)
	voteRepo := repository.NewVoteRepository(db)
	skipRepo := repository.NewSkipRepository(db)
	//创建投票模块的服务
	voteSvc := service.NewVoteService(questionRepo, optionRepo, voteRepo, skipRepo)
	//创建投票模块的handler
	voteHandler := handler.NewVoteHandler(voteSvc)

	//创建跳过作答权的服务与handler
	skipSvc := service.NewSkipService(questionRepo, voteRepo, skipRepo)
	skipHandler := handler.NewSkipHandler(skipSvc)

	//创建发布问题的服务与handler
	expRepo := repository.NewExpRepository(db)
	questionSvc := service.NewQuestionService(questionRepo, optionRepo, voteRepo, skipRepo, userRepo, expRepo)
	questionHandler := handler.NewQuestionHandler(questionSvc)

	//创建评论区的服务与handler
	replyRepo := repository.NewReplyRepository(db)
	replySvc := service.NewReplyService(questionRepo, replyRepo, userRepo)
	replyHandler := handler.NewReplyHandler(replySvc)

	//handler 全部收进一个结构体交给路由，以后每加一个模块只改这一处
	handlers := &handler.Handlers{User: userHandler, Me: meHandler, Vote: voteHandler, Skip: skipHandler, Question: questionHandler, Reply: replyHandler}

	//到期结算：后台每 30 秒扫一轮
	settleSvc := service.NewSettleService(questionRepo, voteRepo, optionRepo)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM) //监听系统信号，收到退出信号就取消ctx
	defer stop()
	go settleSvc.Run(ctx)

	r := router.SetupRouter(handlers)
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("HTTP 服务启动失败: %v", err)
	}
}
