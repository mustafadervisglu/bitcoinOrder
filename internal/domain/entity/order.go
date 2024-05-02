package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Order struct {
	gorm.Model
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID        uuid.UUID `gorm:"type:uuid;not null"`
	Type          string    `gorm:"type:varchar(10);not null"`
	Asset         string    `gorm:"type:varchar(255)"`
	OrderPrice    float64   `gorm:"type:double precision"`
	OrderQuantity float64   `gorm:"type:double precision"`
	OrderStatus   bool      `gorm:"type:boolean;default:true"`
	CreatedAt     time.Time
	CompletedAt   *time.Time `gorm:"default:null"`
	User          Users      `gorm:"foreignKey:UserID"`
}
