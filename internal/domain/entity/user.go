package entity

import (
	"github.com/google/uuid"
	"time"
)

type Users struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Email       string     `gorm:"type:varchar(255);unique;index:idx_users_email"`
	BtcBalance  *float64   `gorm:"type:double precision"`
	UsdtBalance *float64   `gorm:"type:double precision"`
	CreatedAt   time.Time  `gorm:"type:timestamp"`
	UpdatedAt   *time.Time `gorm:"type:timestamp"`
	DeletedAt   *time.Time `gorm:"type:timestamp"`
	Orders      []Order    `gorm:"foreignKey:UserID"`
}
