package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"echo/model"
	"echo/repository"
)

type QuestionService struct {
	questionRepo *repository.QuestionRepository
	optionRepo   *repository.OptionRepository
	voteRepo     *repository.VoteRepository
	skipRepo     *repository.SkipRepository
	userRepo     *repository.UserRepository
	expRepo      *repository.ExpRepository
}

func NewQuestionService(
	questionRepo *repository.QuestionRepository,
	optionRepo *repository.OptionRepository,
	voteRepo *repository.VoteRepository,
	skipRepo *repository.SkipRepository,
	userRepo *repository.UserRepository,
	expRepo *repository.ExpRepository,
) *QuestionService {
	return &QuestionService{
		questionRepo: questionRepo,
		optionRepo:   optionRepo,
		voteRepo:     voteRepo,
		skipRepo:     skipRepo,
		userRepo:     userRepo,
		expRepo:      expRepo,
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
	TotalVotes    int64           `json:"total_votes"`
	CanVote       bool            `json:"can_vote"`
	MyVote        *MyVote         `json:"my_vote"`
	Asker         model.UserBrief `json:"asker"`
	Options       []OptionView    `json:"options"`
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

	// 5.提问者必须存在（token有效但账号已被删的情况）
	asker, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	q := &model.Question{
		AskerID:     userID,
		Description: desc,
		Status:      model.StatusOpen, //必须显式写：零值 "" 会覆盖 DB DEFAULT 并撞 CHECK 约束
		ExpiresAt:   time.Now().Add(time.Duration(req.DurationHours) * time.Hour),
		OptionCount: int16(len(labels)),
	}

	opts, err := s.questionRepo.CreateWithOptions(ctx, q, labels, func(tx *gorm.DB, q *model.Question) error {
		return s.expRepo.GrantTx(tx, &model.ExpEntry{
			UserID:     userID,
			Delta:      model.ExpAsk,
			Reason:     model.ExpReasonAsk,
			SourceType: model.ExpSourceQuestion,
			SourceID:   q.ID, //每题只发一次，重复调用会被唯一约束拦下
		})
	})
	if err != nil {
		return nil, err
	}

	asker.Exp += model.ExpAsk //响应里的等级要反映加分后的结果

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
		Asker:         model.NewUserBrief(asker),        // 替代原来那段 AskerBrief{...}
		Options:       make([]OptionView, 0, len(opts)), //空切片而非nil 保证序列化成[]不是null
	}
	for _, o := range opts {
		detail.Options = append(detail.Options, OptionView{
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

// GetDetail 问题详情与实时占比
func (s *QuestionService) GetDetail(ctx context.Context, questionID, userID int64) (*QuestionDetail, error) {
	//问题不存在直接返回哨兵
	q, err := s.questionRepo.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, q, userID) //所以将共同逻辑提取到 buildDetail，统一处理，避免出现不同接口所返回的值不同
}

// Random 随机抽一题；池子为空返回 nil,nil
func (s *QuestionService) Random(ctx context.Context, userID int64) (*QuestionDetail, error) {
	q, err := s.questionRepo.PickRandom(ctx, userID) //随机取题
	if err != nil || q == nil {
		return nil, err
	} //空池也是正常状态
	return s.buildDetail(ctx, q, userID)
}

// buildDetail 组装对外的题目视图，详情与抽题共用同一套，占比不会出现第二种算法
func (s *QuestionService) buildDetail(ctx context.Context, q *model.Question, userID int64) (*QuestionDetail, error) {
	questionID := q.ID

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

	// 6.can_vote=在窗口内+没投过+没跳过+不是自己的题（IsLive已含未关闭、未过期、未结算）
	canVote := q.IsLive() && myV == nil && !skipped && q.AskerID != userID

	// 7.对外的 open/closed 判定见 outwardStatus，与投票接口共用同一个 IsLive
	status := outwardStatus(q)

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
		TotalVotes:    totalVotes,
		CanVote:       canVote,
		Asker:         model.NewUserBrief(asker),
		Options:       make([]OptionView, 0, len(opts)),
	}
	if myV != nil {
		detail.MyVote = &MyVote{OptionID: myV.OptionID, CreatedAt: myV.CreatedAt}
	}

	//percent 复用投票模块那一个函数，两个接口必须给出一模一样的数字
	for _, o := range opts {
		votes := counts[o.ID]
		detail.Options = append(detail.Options, OptionView{
			ID:      o.ID,
			Label:   o.Label,
			SortNo:  o.SortNo,
			Votes:   votes,
			Percent: percent(votes, totalVotes),
		})
	}

	return detail, nil
}

// outwardStatus 对外的题目状态
func outwardStatus(q *model.Question) string {
	if q.IsLive() {
		return model.StatusOpen
	}
	return model.StatusClosed
}

// contestedListLimit 争议列表固定返回最新 100 条
const contestedListLimit = 100

// ContestedItem 争议列表的一项
type ContestedItem struct {
	ID          int64        `json:"id"`
	Description string       `json:"description"`
	Status      string       `json:"status"`
	ExpiresAt   time.Time    `json:"expires_at"`
	CreatedAt   time.Time    `json:"created_at"`
	SettledAt   *time.Time   `json:"settled_at"`
	ContestedAt *time.Time   `json:"contested_at"`
	TotalVotes  int64        `json:"total_votes"`
	Options     []OptionView `json:"options"`
}

// ContestedList 争议列表响应。returned 是本次条数（上限 100），total 是争议库真实总数
type ContestedList struct {
	List     []ContestedItem `json:"list"`
	Returned int64           `json:"returned"`
	Total    int64           `json:"total"`
}

// Contested 争议海螺列表
func (s *QuestionService) Contested(ctx context.Context) (*ContestedList, error) {
	list, err := s.questionRepo.ListContested(ctx, contestedListLimit) //查询争议问题
	if err != nil {
		return nil, err
	}
	total, err := s.questionRepo.CountContested(ctx) //查询问题总数
	if err != nil {
		return nil, err
	}

	ids := make([]int64, 0, len(list)) //提取这一批的用户ID，批量查询选项和票数
	for i := range list {
		ids = append(ids, list[i].ID)
	}
	opts, err := s.optionRepo.ListByQuestions(ctx, ids) //批量查询选项
	if err != nil {
		return nil, err
	}
	optionIDs := make([]int64, 0, len(opts)) //提取选项ID，批量查询票数
	for _, o := range opts {
		optionIDs = append(optionIDs, o.ID)
	}
	counts, err := s.voteRepo.CountByOptions(ctx, optionIDs) //批量统计票数
	if err != nil {
		return nil, err
	}

	//第一遍按题分组，同时把每题总票数累出来——percent 要等总数出来才能算
	byQuestion := make(map[int64][]OptionView, len(list)) //该问题对应的选项列表
	totals := make(map[int64]int64, len(list)) //该问题的总票数
	
	//组装选项和总票数
	for _, o := range opts { 
		votes := counts[o.ID] //0 票的选项不在 map 里，取出来正好是 0
		totals[o.QuestionID] += votes
		byQuestion[o.QuestionID] = append(byQuestion[o.QuestionID], OptionView{
			ID:     o.ID,
			Label:  o.Label,
			SortNo: o.SortNo,
			Votes:  votes,
		})
	}

	//组装争议问题列表
	items := make([]ContestedItem, 0, len(list))
	for i := range list {
		q := &list[i]
		views := byQuestion[q.ID]
		if views == nil {
			views = []OptionView{}
		}
		for j := range views {
			//占比用那个唯一的 percent 函数，和详情、投票同一个算法
			views[j].Percent = percent(views[j].Votes, totals[q.ID])
		}
		items = append(items, ContestedItem{
			ID:          q.ID,
			Description: q.Description,
			Status:      outwardStatus(q),
			ExpiresAt:   q.ExpiresAt,
			CreatedAt:   q.CreatedAt,
			SettledAt:   q.SettledAt,
			ContestedAt: q.ContestedAt,
			TotalVotes:  totals[q.ID],
			Options:     views,
		})
	}

	return &ContestedList{List: items, Returned: int64(len(items)), Total: total}, nil
}
