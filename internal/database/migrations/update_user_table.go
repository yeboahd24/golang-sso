package migrations

import (
	"gorm.io/gorm"
)

func UpdateUserTable(db *gorm.DB) error {
	return db.Exec(`
        ALTER TABLE users 
        ALTER COLUMN last_login TYPE timestamp without time zone,
        ALTER COLUMN last_login SET NOT NULL,
        ALTER COLUMN last_login SET DEFAULT CURRENT_TIMESTAMP
    `).Error
}
