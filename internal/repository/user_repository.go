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

func (r *userRepository) FindByEmail(email string) (*model.User, error) {
	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil // Return nil, nil for not found - this is not an error case
	}
	return &user, err
}

func (r *userRepository) FindByID(id uint) (*model.User, error) {
	var user model.User
	err := r.db.First(&user, id).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil // Return nil if user not found
	}
	return &user, err
}

// func (r *userRepository) CreateOrUpdate(user *model.User) error {
// 	// Basic validation
// 	if user.Email == "" || user.SSOID == "" {
// 		return fmt.Errorf("email and sso_id are required")
// 	}

// 	// Simple create logic
// 	result := r.db.Create(user)
// 	if result.Error != nil {
// 		return fmt.Errorf("failed to create user: %w", result.Error)
// 	}

// 	return nil
// }

func (r *userRepository) CreateOrUpdate(user *model.User) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if user.ID == 0 {
			// This is a new user
			result := tx.Create(user)
			if result.Error != nil {
				return result.Error
			}
		} else {
			// This is an existing user
			result := tx.Model(user).Updates(map[string]interface{}{
				"email":      user.Email,
				"name":       user.Name,
				"sso_id":     user.SSOID, // Ensure this field is included
				"role":       user.Role,
				"last_login": user.LastLogin,
			})
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}
