package postgresql

import (
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
)

func NewConfig(host, user, password, dbName, port, sslMode, timeZone string) *Config {
	return &Config{
		Host:     host,
		User:     user,
		Password: password,
		DbName:   dbName,
		Port:     port,
		SSLMode:  sslMode,
		TimeZone: timeZone,
	}
}

func OpenDB(config *Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s search_path=public",
		config.Host, config.User, config.Password, config.DbName, config.Port, config.SSLMode, config.TimeZone)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
		return nil
	}
	return db
}
