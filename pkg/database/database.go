package database

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"time"
)

type DBConn struct {
	Pool *sql.DB
}

func NewDBConnection(config *Config) (*gorm.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s search_path=public",
		config.Host, config.User, config.Password, config.DBName, config.Port, config.SSLMode, config.TimeZone)
	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("database connection could not be obtained: %w", err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	log.Println("successfully connected to the database !!")

	return db, nil
}

func (d *DBConn) GetConnection() (*sql.DB, error) {
	if err := d.Pool.Ping(); err != nil {
		return nil, fmt.Errorf("database connection closed, reconnecting: %w", err)
	}
	return d.Pool, nil
}
func (d *DBConn) Close() error {
	return d.Pool.Close()
}
