package repository

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDB 创建数据库连接与配置连接池
func NewDB() (*gorm.DB, error) {
	_ = godotenv.Load()

	// 检查环境变量是否存在
	if os.Getenv("DB_HOST") == "" || os.Getenv("DB_NAME") == "" {
		return nil, fmt.Errorf("缺少 DB_HOST 或 DB_NAME，请检查 .env")
	}

	// 检查其他必要的环境变量
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"))

	// GORM 自己的 SQL 日志。级别还是默认的 Warn，只多做一件事：把"记录不存在"排除掉。
	// "我这题投过没""这个账号在不在"这类查询，查不到是正常状态，不该在日志里刷一行 record not found，
	// 否则它会污染"日志里 500 计数必须为 0"这种自测口径。
	// 真正的 SQL 错误（唯一约束、CHECK、语法错、连不上）和慢查询仍然照常打出来。
	gormLogger := logger.New(
		log.New(os.Stdout, "", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond, //超过 200 毫秒的查询按 Warn 打出来
			LogLevel:                  logger.Warn,
			Colorful:                  false, //Windows 终端会显示 ANSI 转义码，关掉更干净
			IgnoreRecordNotFoundError: true,
		},
	)

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true, Logger: gormLogger}) // 开启错误翻译，翻译成gorm自己的错误，方便后面进行错误断言
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层连接池失败: %w", err)
	}

	// 配置数据库连接池
	sqlDB.SetMaxOpenConns(20)                  //最大连接数为20
	sqlDB.SetMaxIdleConns(5)                   //最大空闲连接数为5
	sqlDB.SetConnMaxLifetime(30 * time.Minute) //最大连接存活时间为30分钟

	// 配置数据库连接池
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库 Ping 失败: %w", err)
	}

	return db, nil
}
