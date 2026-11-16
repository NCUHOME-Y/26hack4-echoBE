package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"echo/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VoteRepository struct {
	db *gorm.DB
}

func NewVoteRepository(db *gorm.DB) *VoteRepository {
	return &VoteRepository{db: db}
}

// Create 创建一条投票记录
func (r *VoteRepository) Create(ctx context.Context, v *model.Vote) error {
	err := r.db.WithContext(ctx).Create(v).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return model.ErrAlreadyVoted
	}
	if err != nil {
		return fmt.Errorf("写入投票失败: %w", err)
	}
	return nil
}

// GetByQuestionAndUser 查询某用户对某题的投票记录，没投过返回 nil
func (r *VoteRepository) GetByQuestionAndUser(ctx context.Context, questionID, userID int64) (*model.Vote, error) {
	var v model.Vote
	err := r.db.WithContext(ctx).Where("question_id = ? AND user_id = ?", questionID, userID).First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询我的投票失败: %w", err)
	}
	return &v, nil
}

// ListByQuestion 取出这道题的全部投票明细，结算发分要用：谁投了哪个选项
func (r *VoteRepository) ListByQuestion(ctx context.Context, questionID int64) ([]model.Vote, error) {
	var list []model.Vote
	err := r.db.WithContext(ctx).Where("question_id = ?", questionID).Order("id").Find(&list).Error
	if err != nil {
		return nil, fmt.Errorf("查询投票明细失败: %w", err)
	}
	return list, nil
}

// CountByOption 统计某题每个选项的票数
func (r *VoteRepository) CountByOption(ctx context.Context, questionID int64) (map[int64]int64, error) {
	type row struct {
		OptionID int64
		Votes    int64
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&model.Vote{}).
		Select("option_id, count(*) AS votes").
		Where("question_id = ?", questionID).
		Group("option_id").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("统计选项票数失败: %w", err)
	}

	counts := make(map[int64]int64, len(rows))
	for _, x := range rows {
		counts[x.OptionID] = x.Votes
	}
	return counts, nil
} //注意：没有票的选项不会出现在数据库聚合结果中，因此 service 层需要给缺失选项补 0

// CountByOptions 一次性统计一批选项的票数
func (r *VoteRepository) CountByOptions(ctx context.Context, optionIDs []int64) (map[int64]int64, error) {
	//空参数处理
	if len(optionIDs) == 0 {
		return map[int64]int64{}, nil
	}
	type row struct {
		OptionID int64
		Votes    int64
	}
	//批量聚合查询
	var rows []row
	err := r.db.WithContext(ctx).Model(&model.Vote{}).
		Select("option_id, count(*) AS votes").
		Where("option_id IN ?", optionIDs).
		Group("option_id").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("批量统计选项票数失败: %w", err)
	}
	//转换为map
	counts := make(map[int64]int64, len(rows))
	for _, x := range rows {
		counts[x.OptionID] = x.Votes
	}
	return counts, nil
}

// WithTx 返回一个绑到事务的仓库副本
func (r *VoteRepository) WithTx(tx *gorm.DB) *VoteRepository {
	return &VoteRepository{db: tx}
}

// Cast 在一个事务里完成"锁题 → 复查窗口 → 查跳过 → 插票 → 发积分"（下面这一块涉及到并发和锁，很大部分都依赖AI了）
// afterVote 由 service 传入，用来在同一个事务里发积分，传 nil 表示不发
func (r *VoteRepository) Cast(ctx context.Context, questionID, userID, optionID int64, skips *SkipRepository, afterVote func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		//锁住这道题，让同一题上的投票请求排队，而不是各查各的快照
		var locked model.Question
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, questionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.ErrQuestionNotFound
			}
			return fmt.Errorf("锁定问题失败: %w", err)
		}

		//锁到手之后再看一眼还在不在作答窗口内
		if !locked.IsLive() {
			return model.ErrQuestionClosed
		}

		//再次检查跳过状态，多并发中容易出现错误
		skipped, err := skips.WithTx(tx).Exists(ctx, questionID, userID)
		if err != nil {
			return err
		}
		if skipped {
			return model.ErrQuestionSkipped
		}

		//插入投票
		if err := r.WithTx(tx).Create(ctx, &model.Vote{
			QuestionID: questionID,
			UserID:     userID,
			OptionID:   optionID,
		}); err != nil {
			return err
		}

		//事务最后发积分：发分失败整笔投票回滚，不留"票记上了、分没给"的状态
		if afterVote != nil {
			return afterVote(tx)
		}
		return nil
	})
}

// CountBetween 统计落在一个区间内的产生的投票数量
func (r *VoteRepository) CountBetween(ctx context.Context, start, end time.Time) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.Vote{}).
		Where("created_at >= ? AND created_at < ?", start, end).
		Count(&n).Error
	if err != nil {
		return 0, fmt.Errorf("统计今日投票量失败: %w", err)
	}
	return n, nil
}
