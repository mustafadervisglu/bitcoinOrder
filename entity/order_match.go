package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type OrderMatch struct {
	gorm.Model
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrderID1  uuid.UUID `gorm:"type:uuid;not null"`
	OrderID2  uuid.UUID `gorm:"type:uuid;not null"`
	MatchedAt time.Time `gorm:"default:current_timestamp"`
}
