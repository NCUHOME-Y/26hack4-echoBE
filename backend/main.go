package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"echo/handler"
	"echo/middleware"
	"echo/repository"
	"echo/router"
	"echo/service"

	"go.uber.org/zap"
)

//shutdownTimeout 关停时给在途请求留的时间，超出这个时间会直接关闭
const shutdownTimeout = 10 * time.Second

func main() {

	zapLogger, err := zap.NewProduction() //创建zap（生产格式：json
	if err != nil {
		//这一条只能留在标准库：zap 自己没起来，没有别的日志器可用
		log.Fatalf("zap日志初始化失败: %v", err)
	}
	defer zapLogger.Sync() //程序退出前把缓冲区里的日志全部刷出去 防丢

	middleware.InitLogger(zapLogger)

	db, err := repository.NewDB(zapLogger) //建立数据库连接，SQL 日志并入同一个 zap
	if err != nil {
		middleware.L().Fatal("数据库初始化失败", zap.Error(err))
	}

	//godotenv 是在 NewDB 里加载的，所以这个检查必须放在它后面
	if os.Getenv("JWT_SECRET") == "" {
		middleware.L().Fatal("缺少 JWT_SECRET，请检查 .env")
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
	expRepo := repository.NewExpRepository(db) //投票发分和发布发分共用同一个仓库，所以在这里建一次
	//创建投票模块的服务
	voteSvc := service.NewVoteService(questionRepo, optionRepo, voteRepo, skipRepo, expRepo)
	//创建投票模块的handler
	voteHandler := handler.NewVoteHandler(voteSvc)

	//创建跳过作答权的服务与handler
	skipSvc := service.NewSkipService(questionRepo, voteRepo, skipRepo)
	skipHandler := handler.NewSkipHandler(skipSvc)

	//创建发布问题的服务与handler
	questionSvc := service.NewQuestionService(questionRepo, optionRepo, voteRepo, skipRepo, userRepo, expRepo)
	questionHandler := handler.NewQuestionHandler(questionSvc)

	//创建评论区的服务与handler
	replyRepo := repository.NewReplyRepository(db)
	replySvc := service.NewReplyService(questionRepo, replyRepo, userRepo)
	replyHandler := handler.NewReplyHandler(replySvc)

	//到期结算与提问者公布共用同一个服务，所以要先建好再交给 handler 结构体
	settleSvc := service.NewSettleService(questionRepo, voteRepo, optionRepo, expRepo)
	finalHandler := handler.NewFinalHandler(settleSvc)

	//今日统计
	statsSvc := service.NewStatsService(questionRepo, voteRepo)
	statsHandler := handler.NewStatsHandler(statsSvc)

	//handler 全部收进一个结构体交给路由，以后每加一个模块只改这一处
	handlers := &handler.Handlers{User: userHandler, Me: meHandler, Vote: voteHandler, Skip: skipHandler, Question: questionHandler, Reply: replyHandler, Final: finalHandler, Stats: statsHandler}

	//到期结算：后台每 30 秒扫一轮
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM) //监听系统信号，收到退出信号就取消ctx
	defer stop()
	go settleSvc.Run(ctx)

	r := router.SetupRouter(handlers)
	port := os.Getenv("PORT") //监听端口可由部署环境注入
	if port == "" {
		port = "8080" //不设就保持原来的写死行为，避免改代码顺带改端口
	}

	//创建 HTTP 服务启动
	srv := &http.Server{Addr: ":" + port, Handler: r}

	//容量为1的错误通道：ListenAndServe 退出时要发送结果，而此时主协程可能已经在收尾了，不带缓冲会让这个协程永久卡在发送上
	serverErr := make(chan error, 1)

	//用goroutine启动 HTTP 服务，主协程继续往下走，等待退出信号
	go func() {
		err := srv.ListenAndServe() //监听端口，接收请求，交给gin处理，一直运行到服务停止
		if errors.Is(err, http.ErrServerClosed) {
			err = nil //ErrServerClosed 是 Shutdown 的正常收尾，不是故障
		}
		serverErr <- err
	}()

	select {
	case err := <-serverErr:
		//端口被占、地址非法这类启动失败发生时还没收到任何信号，必须立刻退出，
		//不能留下一个"进程还在但没在服务"的状态
		if err != nil {
			middleware.L().Fatal("HTTP 服务启动失败", zap.Error(err))
		}
	case <-ctx.Done(): //Ctrl+C 或 SIGTERM 到达，开始收尾
	}

	//ctx 已被信号取消，结算 ticker 同时也就停了，这里只负责把在途请求放干净
	stop()
	middleware.L().Info("收到退出信号，停止接收新请求", zap.Duration("grace", shutdownTimeout))

	//优雅退出 HTTP 服务：给在途请求留点时间，超过这个时间还没结束的就直接退出
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		middleware.L().Error("在途请求未在期限内结束，直接退出", zap.Duration("grace", shutdownTimeout), zap.Error(err))
		return
	}
	middleware.L().Info("HTTP 服务已优雅退出")
}
