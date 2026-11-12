package repository

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true}) // 开启错误翻译，翻译成gorm自己的错误，方便后面进行错误断言
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层连接池失败: %w", err)
	}

	// 配置数据库连接池
	sqlDB.SetMaxOpenConns(20) //最大连接数为20
	sqlDB.SetMaxIdleConns(5) //最大空闲连接数为5
	sqlDB.SetConnMaxLifetime(30 * time.Minute) //最大连接存活时间为30分钟

	// 配置数据库连接池
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("数据库 Ping 失败: %w", err)
	}

	return db, nil
}
