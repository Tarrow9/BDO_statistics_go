package repo

import (
	"errors"
	"sync"

	"bdo_calc_go/internal/model"
)

var ErrNotFound = errors.New("not found")

// 인터페이스
type UserRepo interface {
	Save(u *model.User) error
	FindByID(id string) (*model.User, error)
	List() ([]*model.User, error)
}

type userRepoInMemory struct {
	mu    sync.RWMutex
	store map[string]*model.User
}

func NewUserRepoInMemory() UserRepo {
	return &userRepoInMemory{store: make(map[string]*model.User)}
}

func (r *userRepoInMemory) Save(u *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.store[u.ID] = u
	return nil
}

func (r *userRepoInMemory) FindByID(id string) (*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.store[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (r *userRepoInMemory) List() ([]*model.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*model.User, 0, len(r.store))
	for _, u := range r.store {
		out = append(out, u)
	}
	return out, nil
}
