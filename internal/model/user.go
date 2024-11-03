package model

import (
	"gorm.io/gorm"
	"time"
)

type User struct {
	gorm.Model
	Email     string `gorm:"unique;not null"`
	Name      string
	LastLogin time.Time
}
