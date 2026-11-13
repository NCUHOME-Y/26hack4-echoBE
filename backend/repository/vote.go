package repository

import (
	"context"
	"errors"
	"fmt"

	"echo/model"

	"gorm.io/gorm"
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
