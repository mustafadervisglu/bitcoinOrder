package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Users struct {
	gorm.Model
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Email      string    `gorm:"type:varchar(255);unique;not null"`
	BtcBalance *float64  `gorm:"type:double precision"`
	UsdBalance *float64  `gorm:"type:double precision"`
	Orders     []Order   `gorm:"many2many:user_orders;"`
}
