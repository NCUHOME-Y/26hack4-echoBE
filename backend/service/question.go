package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"echo/model"
	"echo/repository"
)

type QuestionService struct {
	questionRepo *repository.QuestionRepository
	userRepo     *repository.UserRepository
}

func NewQuestionService(qr *repository.QuestionRepository, ur *repository.UserRepository) *QuestionService {
	return &QuestionService{questionRepo: qr, userRepo: ur}
}

// questiondetail发布/详情接口的响应体
type QuestionDetail struct {
	ID            int64           `json:"id"`
	Description   string          `json:"description"`
	Status        string          `json:"status"`
	ExpiresAt     time.Time       `json:"expires_at"`
	CreatedAt     time.Time       `json:"created_at"`
	SettledAt     *time.Time      `json:"settled_at"`
	ContestedAt   *time.Time      `json:"contested_at"`
	FinalOptionID *int64          `json:"final_option_id"`
	OptionCount   int16           `json:"option_count"`
	TotalVotes    int             `json:"total_votes"`
	CanVote       bool            `json:"can_vote"`
	MyVote        *MyVote         `json:"my_vote"`
	Asker         model.UserBrief `json:"asker"`
	Options       []OptionBrief   `json:"options"`
}

type OptionBrief struct {
	ID      int64   `json:"id"`
	Label   string  `json:"label"`
	SortNo  int16   `json:"sort_no"`
	Votes   int     `json:"votes"`
	Percent float64 `json:"percent"`
}

type MyVote struct {
	OptionID  int64     `json:"option_id"`
	CreatedAt time.Time `json:"created_at"`
}

// create发布问题：校验，入库，组装详情响应体
func (s *QuestionService) Create(ctx context.Context, userID int64, req model.CreateQuestionReq) (*QuestionDetail, error) {
	// 1.描述:trim后1~100字符
	desc := strings.TrimSpace(req.Description)
	if n := utf8.RuneCountInString(desc); n < 1 || n > 100 {
		return nil, model.ErrInvalidDescription
	}

	// 2.选项数量2~4
	if len(req.Options) < 2 || len(req.Options) > 4 {
		return nil, model.ErrInvalidOptionCount
	}

	// 3.每个选项文字1~18字符
	labels := make([]string, 0, len(req.Options))
	for _, o := range req.Options {
		label := strings.TrimSpace(o.Label)
		if n := utf8.RuneCountInString(label); n < 1 || n > 18 {
			return nil, model.ErrInvalidOptionLabel
		}
		labels = append(labels, label)
	}

	// 4.结算档位只能是六个值
	if !validDuration(req.DurationHours) {
		return nil, model.ErrInvalidDuration
	}

	// 5.提问者必须存在（token有效但账号被删的情况）（问的ai,我没想到这个
	asker, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	q := &model.Question{
		AskerID:     userID,
		Description: desc,
		Status:      "open", //问的ai:必须显式写：零值 "" 会覆盖 DB DEFAULT 并撞 CHECK 约束
		ExpiresAt:   time.Now().Add(time.Duration(req.DurationHours) * time.Hour),
		OptionCount: int16(len(labels)),
	}

	opts, err := s.questionRepo.CreateWithOptions(ctx, q, labels)
	if err != nil {
		return nil, err
	}

	detail := &QuestionDetail{
		ID:            q.ID,
		Description:   q.Description,
		Status:        q.Status,
		ExpiresAt:     q.ExpiresAt,
		CreatedAt:     q.CreatedAt,
		SettledAt:     q.SettledAt,
		ContestedAt:   q.ContestedAt,
		FinalOptionID: q.FinalOptionID,
		OptionCount:   q.OptionCount,
		TotalVotes:    0,     //刚发布默认投票数为0
		CanVote:       false, //自己的题不能投
		MyVote:        nil,
		Asker:         model.NewUserBrief(asker),         // 替代原来那段 AskerBrief{...}
		Options:       make([]OptionBrief, 0, len(opts)), //空切片而非nil 保证序列化成[]不是null
	}
	for _, o := range opts {
		detail.Options = append(detail.Options, OptionBrief{
			ID:      o.ID,
			Label:   o.Label,
			SortNo:  o.SortNo,
			Votes:   0,
			Percent: 0,
		})
	}
	return detail, nil
}
func validDuration(h int) bool {
	switch h {
	case 1, 3, 6, 12, 24, 48:
		return true
	}
	return false
}
