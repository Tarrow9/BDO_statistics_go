package service

import (
	"errors"

	"bdo_calc_go/internal/model"
	"bdo_calc_go/internal/repo"
	"bdo_calc_go/pkg/logger"
	// "github.com/google/uuid"
)

type UserService struct {
	repo   repo.UserRepo
	logger logger.Logger
}

func NewUserService(r repo.UserRepo, l logger.Logger) *UserService {
	return &UserService{repo: r, logger: l}
}

func (s *UserService) Create(name, email string) (*model.User, error) {
	if name == "" || email == "" {
		return nil, errors.New("name and email are required")
	}
	u := &model.User{
		ID:    name,
		Name:  name,
		Email: email,
	}
	if err := s.repo.Save(u); err != nil {
		return nil, err
	}
	s.logger.Infof("user created: %s", u.ID)
	return u, nil
}

func (s *UserService) GetByID(id string) (*model.User, error) {
	return s.repo.FindByID(id)
}

func (s *UserService) List() ([]*model.User, error) {
	return s.repo.List()
}
