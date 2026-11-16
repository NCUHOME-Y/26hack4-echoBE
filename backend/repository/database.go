package repository

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDB 创建数据库连接与配置连接池。SQL 日志走传进来的 zap，服务里只保留一种日志格式
func NewDB(l *zap.Logger) (*gorm.DB, error) {
	_ = godotenv.Load()

	// 检查环境变量是否存在
	if os.Getenv("DB_HOST") == "" || os.Getenv("DB_NAME") == "" {
		return nil, fmt.Errorf("缺少 DB_HOST 或 DB_NAME，请检查 .env")
	}

	// 检查其他必要的环境变量
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_SSLMODE"))

	// GORM 的 SQL 日志接到 zap 上，级别 Warn：只出真错误和慢查询
	gormLogger := newGormZap(l, 200*time.Millisecond, logger.Warn)

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
