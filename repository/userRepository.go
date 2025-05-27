package repository

import (
	"context"
	"tx-demo/model"
)

type UserRepository interface {
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	CreateUser(ctx context.Context, user *model.User) error
	FindByUserID(ctx context.Context, userID string) (*model.User, error)
}

type userRepository struct {
	*Repository
}

func NewUserRepository(
	r *Repository,
) UserRepository {
	return &userRepository{
		Repository: r,
	}
}

// FindByUsername 根据用户名查询用户
func (u *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	return &user, u.DB(ctx).Where("username = ?", username).First(&user).Error
}

// CreateUser 创建新用户
func (u *userRepository) CreateUser(ctx context.Context, user *model.User) error {
	return u.DB(ctx).Create(user).Error
}

// FindByUserID 根据 用户ID 查询用户
func (u *userRepository) FindByUserID(ctx context.Context, userID string) (*model.User, error) {
	var user model.User
	return &user, u.DB(ctx).Where("user_id = ?", userID).First(&user).Error
}
