package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// gormZap 把 GORM 的 SQL 日志搬进 zap
type gormZap struct {
	l     *zap.Logger
	level logger.LogLevel
	slow  time.Duration
}

func newGormZap(l *zap.Logger, slow time.Duration, level logger.LogLevel) logger.Interface {
	return &gormZap{l: l, level: level, slow: slow}
}

func (g *gormZap) LogMode(level logger.LogLevel) logger.Interface {
	clone := *g
	clone.level = level
	return &clone
}

func (g *gormZap) Info(ctx context.Context, format string, args ...any) {
	if g.level >= logger.Info {
		g.l.Info(fmt.Sprintf(format, args...))
	}
}

func (g *gormZap) Warn(ctx context.Context, format string, args ...any) {
	if g.level >= logger.Warn {
		g.l.Warn(fmt.Sprintf(format, args...))
	}
}

func (g *gormZap) Error(ctx context.Context, format string, args ...any) {
	if g.level >= logger.Error {
		g.l.Error(fmt.Sprintf(format, args...))
	}
}

// Trace 每条执行过的 SQL 都会经过这里
func (g *gormZap) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if g.level <= logger.Silent {
		return
	}
	//查不到是正常状态，不是故障：像"我这题投过没""这个账号在不在"都是靠 not found 判定的，
	//留下来的话会把"error 行数必须为 0"这个自测口径污染掉
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return
	}

	sql, rows := fc()
	elapsed := time.Since(begin)

	switch {
	case err != nil && g.level >= logger.Error:
		g.l.Error("SQL 执行失败",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
			zap.Error(err),
		)
	case g.slow > 0 && elapsed > g.slow && g.level >= logger.Warn:
		g.l.Warn("慢查询",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
		)
	case g.level >= logger.Info:
		g.l.Info("SQL",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Duration("elapsed", elapsed),
		)
	}
}
