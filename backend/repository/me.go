package repository

import (
	"context"
	"fmt"

	"echo/model"

	"gorm.io/gorm"
)

type MeRepository struct {
	db *gorm.DB
}

func NewMeRepository(db *gorm.DB) *MeRepository {
	return &MeRepository{db: db}
}

// ListMyQuestions 查询我发布的问题，并通过标量子查询统计每题总票数
func (r *MeRepository) ListMyQuestions(
	ctx context.Context,
	userID int64,
	limit, offset int,
) ([]model.MyQuestionItem, int64, error) {
	list := make([]model.MyQuestionItem, 0)
	var total int64

	if err := r.db.WithContext(ctx).
		Model(&model.Question{}).
		Where("asker_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计我发布的问题失败: %w", err)
	}

	err := r.db.WithContext(ctx).
		Table("questions AS q").
		Select(`
			q.id,
			q.description,
			q.status,
			q.expires_at,
			q.created_at,
			q.settled_at,
			q.contested_at,
			(SELECT COUNT(v.id)
			 FROM votes AS v
			 WHERE v.question_id = q.id) AS total_votes
		`).
		Where("q.asker_id = ?", userID).
		Order("q.created_at DESC, q.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&list).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询我发布的问题失败: %w", err)
	}

	return list, total, nil
}

// ListMyAnswers 查询我投过票的问题 可以按问题文案进行关键词搜索
func (r *MeRepository) ListMyAnswers(
	ctx context.Context,
	userID int64,
	keyword string,
	limit, offset int,
) ([]model.MyAnswerItem, int64, error) {
	list := make([]model.MyAnswerItem, 0)
	var total int64

	// Count 和列表共用筛选条件，避免总数与列表条件不一致。
	baseQuery := func() *gorm.DB {
		query := r.db.WithContext(ctx).
			Table("votes AS v").
			Joins("JOIN questions AS q ON q.id = v.question_id").
			Where("v.user_id = ?", userID)

		if keyword != "" {
			query = query.Where(
				"q.description ILIKE ?",
				"%"+keyword+"%",
			)
		}
		return query
	}

	if err := baseQuery().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计我参与的问题失败: %w", err)
	}

	err := baseQuery().
		Joins("JOIN options AS o ON o.id = v.option_id AND o.question_id = q.id").
		Select(`
			q.id AS question_id,
			q.description,
			q.status,
			v.option_id AS my_option_id,
			o.label AS my_option_label,
			q.final_option_id,
			CASE
				WHEN q.final_option_id IS NULL THEN NULL
				ELSE v.option_id = q.final_option_id
			END AS matched,
			v.created_at AS voted_at,
			q.settled_at
		`).
		Order("v.created_at DESC, v.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&list).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询我参与的问题失败: %w", err)
	}

	return list, total, nil
}
