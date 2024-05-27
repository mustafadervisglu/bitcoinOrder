package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type OrderMatch struct {
	gorm.Model
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	OrderID1      uuid.UUID `gorm:"type:uuid;not null;index:idx_order_match_order_ids"`
	OrderID2      uuid.UUID `gorm:"type:uuid;not null;index:idx_order_match_order_ids"`
	OrderQuantity float64   `gorm:"type:double precision"`
	MatchedAt     time.Time `gorm:"default:current_timestamp"`
}
