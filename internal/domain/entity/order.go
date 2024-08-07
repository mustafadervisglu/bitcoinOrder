package entity

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Order struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;index"`
	Type          string    `gorm:"type:varchar(10);not null;index:idx_order_type"`
	Asset         string    `gorm:"type:varchar(255);index"`
	OrderPrice    float64   `gorm:"type:double precision;index:idx_order_price_type_status"`
	OrderQuantity float64   `gorm:"type:double precision"`
	OrderStatus   bool      `gorm:"type:boolean;default:true;index:idx_order_status"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
	CompletedAt   *time.Time     `gorm:"default:NULL"`
	User          Users          `gorm:"foreignKey:UserID"`
}
