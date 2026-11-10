package main

import (
	"log"
	"os"

	"echo/handler"
	"echo/repository"
	"echo/router"
	"echo/service"
)

func main() {
	db, err := repository.NewDB() //建立数据库连接
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	//godotenv 是在 NewDB 里加载的，所以这个检查必须放在它后面
	if os.Getenv("JWT_SECRET") == "" {
		log.Fatal("缺少 JWT_SECRET，请检查 .env")
	}

	userRepo := repository.NewUserRepository(db) //创建用户仓库
	userSvc := service.NewUserService(userRepo) //创建用户服务
	userHandler := handler.NewUserHandler(userSvc) //创建用户处理器（这里三个是单向向下的依赖关系）

	r := router.SetupRouter(userHandler)
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("HTTP 服务启动失败: %v", err)
	}
}