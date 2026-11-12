package service

import "echo/repository"

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo} //把reposiory层的东西保存到service层中，方便本层调用
}