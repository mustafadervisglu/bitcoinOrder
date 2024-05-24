package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Users struct {
	gorm.Model
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Email       string    `gorm:"type:varchar(255);unique;index:idx_users_email"`
	BtcBalance  *float64  `gorm:"type:double precision"`
	UsdtBalance *float64  `gorm:"type:double precision"`
	Orders      []Order   `gorm:"foreignKey:UserID"`
}
