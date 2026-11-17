package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"echo/model"

	"gorm.io/gorm"
)

type ReplyRepository struct {
	db *gorm.DB
}

func NewReplyRepository(db *gorm.DB) *ReplyRepository {
	return &ReplyRepository{db: db}
}

// Create 写入一条评论
func (r *ReplyRepository) Create(ctx context.Context, rp *model.Reply) error {
	if err := r.db.WithContext(ctx).Create(rp).Error; err != nil {
		return fmt.Errorf("写入评论失败: %w", err)
	}
	return nil
}

// CountByQuestion 该题未软删的评论总条数
func (r *ReplyRepository) CountByQuestion(ctx context.Context, questionID int64) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Reply{}).
		Where("question_id = ? AND deleted_at IS NULL", questionID). //删除时间为空代表没有被软删除
		Count(&n).Error //返回一个类型为int64的数字
	if err != nil {
		return 0, fmt.Errorf("统计评论条数失败: %w", err)
	}
	return n, nil
}

// ListByQuestion 查询某题的评论列表，按时间倒序（最新评论优先），limit 限制条数
func (r *ReplyRepository) ListByQuestion(ctx context.Context, questionID int64, limit int) ([]model.Reply, error) {
	var list []model.Reply
	err := r.db.WithContext(ctx).
		Where("question_id = ? AND deleted_at IS NULL", questionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&list).Error
	if err != nil {
		return nil, fmt.Errorf("查询评论失败: %w", err)
	}
	return list, nil
}

// GetActiveByID根据评论ID取一条还没有被软删除的评论
func (r *ReplyRepository) GetActiveByID(ctx context.Context, id int64) (*model.Reply, error) {
	var rp model.Reply
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id). //评论ID匹配且没有删除时间
		First(&rp).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询评论失败: %w", err)
	}
	return &rp, nil
}

// SoftDelete 给评论盖上删除时间
func (r *ReplyRepository) SoftDelete(ctx context.Context, id int64) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.Reply{}).
		Where("id = ? AND deleted_at IS NULL", id). //防止重复删除
		Update("deleted_at", time.Now()) //给评论增加删除时间，但并不会在数据库里面真正删除这条评论
	if res.Error != nil {
		return false, fmt.Errorf("删除评论失败: %w", res.Error)
	}
	return res.RowsAffected > 0, nil
}
