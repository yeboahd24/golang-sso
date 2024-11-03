package repository

import (
	"github.com/yeboahd24/sso/internal/model"
	"gorm.io/gorm"
)

type UserRepository interface {
	CreateOrUpdate(user *model.User) error
	FindByEmail(email string) (*model.User, error)
	FindByID(id uint) (*model.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateOrUpdate(user *model.User) error {
	return r.db.Where(model.User{Email: user.Email}).
		Assign(user).
		FirstOrCreate(user).Error
}

func (r *userRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *userRepository) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	return &user, err
}
