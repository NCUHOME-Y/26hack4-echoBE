package repository

import (
	"context"
	"errors"
	"fmt"

	"echo/model"

	"gorm.io/gorm"
)

type QuestionRepository struct {
	db *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

// GetByID根据问题ID 查询问题
func (r *QuestionRepository) GetByID(ctx context.Context, id int64) (*model.Question, error) {
	var q model.Question
	err := r.db.WithContext(ctx).First(&q, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrQuestionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询问题失败: %w", err)
	}
	return &q, nil
}

// CreateWithOptions问题与选项在同一事务内插入 失败整体回滚
func (r *QuestionRepository) CreateWithOptions(ctx context.Context, q *model.Question, labels []string) ([]model.Option, error) {
	var opts []model.Option
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(q).Error; err != nil {
			return fmt.Errorf("创建问题失败: %w", err)
		}
		//每次进闭包都重建，事务重试时不会累积出重复选项
		opts = make([]model.Option, 0, len(labels))
		for i, label := range labels {
			opts = append(opts, model.Option{
				QuestionID: q.ID,
				Label:      label,
				SortNo:     int16(i + 1),
			})
		}
		if err := tx.Create(&opts).Error; err != nil {
			return fmt.Errorf("创建选项失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return opts, nil
}

// PickRandom 随机抽一题。池子=未关闭、未过期、未结算，再排除我已投过的、已跳过的、我自己提的。
func (r *QuestionRepository) PickRandom(ctx context.Context, userID int64) (*model.Question, error) {
	var qs []model.Question
	err := r.db.WithContext(ctx).Model(&model.Question{}).
		Where("status = ? AND expires_at > now() AND settled_at IS NULL", model.StatusOpen). //只查询开放且未过期的问题
		Where("asker_id <> ?", userID). //排除用户自己的问题
		Where("id NOT IN (?)", r.db.WithContext(ctx).Model(&model.Vote{}).
			Select("question_id").Where("user_id = ?", userID)). //排除已经投票的问题
		Where("id NOT IN (?)", r.db.WithContext(ctx).Model(&model.QuestionSkip{}).
			Select("question_id").Where("user_id = ?", userID)). //排除已经跳过的问题
		Order("random()").Limit(1). //随机排序取一条
		Find(&qs).Error
	if err != nil {
		return nil, fmt.Errorf("随机抽题失败: %w", err)
	}
	if len(qs) == 0 {
		return nil, nil
	} //空池也是允许的，不是错误
	return &qs[0], nil
}
