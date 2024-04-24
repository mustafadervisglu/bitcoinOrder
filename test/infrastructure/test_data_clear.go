package infrastructure

import (
	"github.com/labstack/gommon/log"
	"gorm.io/gorm"
)

func TestDataClear(db *gorm.DB) {
	truncateRes := db.Exec("TRUNCATE order_matches, orders, users RESTART IDENTITY CASCADE")
	if truncateRes.Error != nil {
		log.Error(truncateRes.Error)
	} else {
		log.Info("Users balances table truncated")
	}
}
