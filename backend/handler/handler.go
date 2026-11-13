package handler

import "echo/service"

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

//把所有 handler 统一放在一个 struct 里，未来要增加只要扩展就好了，不用更改路由函数签名
type Handlers struct {
	User *UserHandler
	Me   *MeHandler
	Vote *VoteHandler
	User     *UserHandler
	Me       *MeHandler
	Question *QuestionHandler
}
