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
	return r.SettleWith(ctx, questionID, finalOptionID, contested, nil)
}

// SettleWith 结算，并在同一个事务里执行 afterSettle（结算积分就从这里发）
func (r *QuestionRepository) SettleWith(ctx context.Context, questionID int64, finalOptionID *int64, contested bool, afterSettle func(tx *gorm.DB) error) (bool, error) {
	var won bool //是否抢到结算权
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updated, err := markSettledTx(tx, questionID, finalOptionID, contested)
		if err != nil || !updated { //没抢到就不用往下走
			return err
		}
		won = true
		if afterSettle != nil {
			return afterSettle(tx)
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return won, nil
}

// markSettledTx 结算写入的实体，绑在调用方给的事务上
func markSettledTx(tx *gorm.DB, questionID int64, finalOptionID *int64, contested bool) (bool, error) {
	fields := map[string]interface{}{
		"status":          model.StatusClosed,
		"settled_at":      time.Now(),
		"final_option_id": finalOptionID,
	} //所需要更新的三个字段
	if contested {
		fields["contested_at"] = time.Now()
	} //不是争议就不写这个字段，让它保持空
	res := tx.Model(&model.Question{}).
		Where("id = ? AND settled_at IS NULL", questionID). //防止被重复结算
		Updates(fields)                                     //在数据库里面更新状态
	if res.Error != nil {
		return false, fmt.Errorf("结算问题失败: %w", res.Error)
	} //是否抢到结算权
	return res.RowsAffected > 0, nil //影响 0 行说明被别人抢先结算过了
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

// CountCreatedBetween 统计落在一个区间内的创建问题数量
func (r *QuestionRepository) CountCreatedBetween(ctx context.Context, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Question{}).
		Where("created_at >= ? AND created_at < ?", start, end).
		Count(&n).Error
	if err != nil {
		return 0, fmt.Errorf("统计今日发布量失败: %w", err)
	}
	return n, nil
}

// CountSettledBetween 统计落在一个区间内的结算问题数量
func (r *QuestionRepository) CountSettledBetween(ctx context.Context, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Question{}).
		Where("settled_at >= ? AND settled_at < ?", start, end).
		Count(&n).Error
	if err != nil {
		return 0, fmt.Errorf("统计今日结算量失败: %w", err)
	}
	return n, nil
}
