package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Lock struct {
	ID        uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null"`
	Asset     string    `gorm:"type:varchar(10);not null"`
	Amount    float64   `gorm:"type:decimal(18,8);not null"`
	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
