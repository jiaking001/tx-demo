package repository

import "tx-demo/model"

type UserRepository interface {
	FindByUsername(username string) (*model.User, error)
	CreateUser(user *model.User) error
	FindByUserID(userID string) (*model.User, error)
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
func (u *userRepository) FindByUsername(username string) (*model.User, error) {
	var user model.User
	return &user, u.db.Where("username = ?", username).First(&user).Error
}

// CreateUser 创建新用户
func (u *userRepository) CreateUser(user *model.User) error {
	return u.db.Create(user).Error
}

// FindByUserID 根据 用户ID 查询用户
func (u *userRepository) FindByUserID(userID string) (*model.User, error) {
	var user model.User
	return &user, u.db.Where("user_id = ?", userID).First(&user).Error
}
