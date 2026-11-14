package service

import (
	"context"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"echo/model"
	"echo/repository"
)

type QuestionService struct {
	questionRepo *repository.QuestionRepository
	optionRepo   *repository.OptionRepository
	voteRepo     *repository.VoteRepository
	skipRepo     *repository.SkipRepository
	userRepo     *repository.UserRepository
}

func NewQuestionService(
	questionRepo *repository.QuestionRepository,
	optionRepo *repository.OptionRepository,
	voteRepo *repository.VoteRepository,
	skipRepo *repository.SkipRepository,
	userRepo *repository.UserRepository,
) *QuestionService {
	return &QuestionService{
		questionRepo: questionRepo,
		optionRepo:   optionRepo,
		voteRepo:     voteRepo,
		skipRepo:     skipRepo,
		userRepo:     userRepo,
	}
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

// getDetail获取问题详情和实时占比
func (s *QuestionService) GetDetail(ctx context.Context, questionID, userID int64) (*QuestionDetail, error) {
	// 1.问题不存在直接返回哨兵
	q, err := s.questionRepo.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}

	// 2.选项按 sort_no 升序
	opts, err := s.optionRepo.ListByQuestion(ctx, questionID)
	if err != nil {
		return nil, err
	}

	// 3.每个选项的票数：没票的选项不出现在map里 下面遍历opts时要补0
	counts, err := s.voteRepo.CountByOption(ctx, questionID)
	if err != nil {
		return nil, err
	}
	var totalVotes int64
	for _, c := range counts {
		totalVotes += c
	}

	// 4.我投过没、跳过没
	myV, err := s.voteRepo.GetByQuestionAndUser(ctx, questionID, userID)
	if err != nil {
		return nil, err
	}
	skipped, err := s.skipRepo.Exists(ctx, questionID, userID)
	if err != nil {
		return nil, err
	}

	// 5.提问者资料
	asker, err := s.userRepo.GetByID(ctx, q.AskerID)
	if err != nil {
		return nil, err
	}

	// 6.can_vote=在窗口内+没投过+没跳过+不是自己的题（IsLive已含未关闭、未过期）
	canVote := q.IsLive() && myV == nil && !skipped && q.AskerID != userID

	// 7.对外只暴露open/closed，判据与投票接口共用IsLive：不在作答窗口内（已关闭、已过期）一律closed，
	status := model.StatusOpen
	if !q.IsLive() {
		status = model.StatusClosed
	}

	detail := &QuestionDetail{
		ID:            q.ID,
		Description:   q.Description,
		Status:        status,
		ExpiresAt:     q.ExpiresAt,
		CreatedAt:     q.CreatedAt,
		SettledAt:     q.SettledAt,
		ContestedAt:   q.ContestedAt,
		FinalOptionID: q.FinalOptionID,
		OptionCount:   q.OptionCount,
		TotalVotes:    int(totalVotes),
		CanVote:       canVote,
		Asker:         model.NewUserBrief(asker),
		Options:       make([]OptionBrief, 0, len(opts)),
	}
	if myV != nil {
		detail.MyVote = &MyVote{OptionID: myV.OptionID, CreatedAt: myV.CreatedAt}
	}

	for _, o := range opts {
		votes := counts[o.ID]
		var percent float64
		if totalVotes > 0 {
			percent = math.Round(float64(votes)/float64(totalVotes)*1000) / 10
			//这个是问ai怎么转换的：53.3333 -> 533.33 -> Round 533 -> /10 = 53.3
		}
		detail.Options = append(detail.Options, OptionBrief{
			ID:      o.ID,
			Label:   o.Label,
			SortNo:  o.SortNo,
			Votes:   int(votes),
			Percent: percent,
		})
	}

	return detail, nil
}
