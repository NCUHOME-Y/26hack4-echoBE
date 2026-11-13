package repository

import (
	"context"
	"fmt"

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
