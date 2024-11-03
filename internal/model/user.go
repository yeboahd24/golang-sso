package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email     string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	Name      string    `gorm:"type:varchar(255)"`
	SSOID     string    `gorm:"column:sso_id;not null"`
	Role      string    `gorm:"type:varchar(50)"`
	LastLogin time.Time `gorm:"type:timestamp;not null"`
	CreatedAt time.Time `gorm:"type:timestamp;not null"`
	UpdatedAt time.Time `gorm:"type:timestamp;not null"`
	DeletedAt gorm.DeletedAt
}
