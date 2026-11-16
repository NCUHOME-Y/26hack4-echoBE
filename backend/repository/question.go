package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

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
// afterCreate 由 service 传入，用来在同一个事务里发积分，传 nil 表示不发
func (r *QuestionRepository) CreateWithOptions(ctx context.Context, q *model.Question, labels []string, afterCreate func(tx *gorm.DB, q *model.Question) error) ([]model.Option, error) {
	var opts []model.Option
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(q).Error; err != nil {
			return fmt.Errorf("创建问题失败: %w", err)
		}
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
		//事务最后发积分：积分失败则整道题回滚，不出现题在分没发
		if afterCreate != nil {
			if err := afterCreate(tx, q); err != nil {
				return err
			}
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
		Where("asker_id <> ?", userID).                                                      //排除用户自己的问题
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

// ListSettleable 捞出到期了但还没结算的题，交给结算函数逐题处理。
func (r *QuestionRepository) ListSettleable(ctx context.Context, limit int) ([]model.Question, error) {
	var list []model.Question
	err := r.db.WithContext(ctx).
		Where("expires_at < now() AND settled_at IS NULL"). //同时满足问题已经到了结算时间，但是没有结算
		Order("expires_at").                                //最早过期，最先处理
		Limit(limit).                                       //避免一次性处理大量问题
		Find(&list).Error
	if err != nil {
		return nil, fmt.Errorf("查询待结算问题失败: %w", err)
	}
	return list, nil
}

// MarkSettled负责结算问题
func (r *QuestionRepository) MarkSettled(ctx context.Context, questionID int64, finalOptionID *int64, contested bool) (bool, error) {
	fields := map[string]interface{}{
		"status":          model.StatusClosed,
		"settled_at":      time.Now(),
		"final_option_id": finalOptionID,
	} //所需要更新的三个字段
	if contested {
		fields["contested_at"] = time.Now()
	} //不是争议就不写这个字段，让它保持空
	res := r.db.WithContext(ctx).Model(&model.Question{}).
		Where("id = ? AND settled_at IS NULL", questionID). //防止被重复结算
		Updates(fields)                                     //在数据库里面更新状态
	if res.Error != nil {
		return false, fmt.Errorf("结算问题失败: %w", res.Error)
	} //是否抢到结算权
	return res.RowsAffected > 0, nil //影响 0 行说明被别人抢先结算过了
}

// MarkFinalOnly 补写最终选择（遇到平票或者没有人投票的时候）
func (r *QuestionRepository) MarkFinalOnly(ctx context.Context, questionID, optionID int64) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.Question{}).
		Where("id = ? AND final_option_id IS NULL", questionID).     //防止重复公布把选择改掉
		Updates(map[string]interface{}{"final_option_id": optionID}) //只修改最终选项
	if res.Error != nil {
		return false, fmt.Errorf("写入最终选择失败: %w", res.Error)
	}
	return res.RowsAffected > 0, nil //影响 0 行说明这题已经定过一次了
}

// ListContested 争议海螺列表：contested_at 非空，按进入争议库的时间倒序取前 limit 条
func (r *QuestionRepository) ListContested(ctx context.Context, limit int) ([]model.Question, error) {
	var list []model.Question
	//查询已经进入争议库的问题
	err := r.db.WithContext(ctx).
		Where("contested_at IS NOT NULL").
		Order("contested_at DESC"). //最新的问题排在最前面
		Limit(limit). //限制查询数量
		Find(&list).Error
	if err != nil {
		return nil, fmt.Errorf("查询争议问题失败: %w", err)
	}
	return list, nil
}

// CountContested 争议库的真实总条数
func (r *QuestionRepository) CountContested(ctx context.Context) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Question{}).
		Where("contested_at IS NOT NULL").
		Count(&n).Error
	if err != nil {
		return 0, fmt.Errorf("统计争议问题失败: %w", err)
	}
	return n, nil
}
