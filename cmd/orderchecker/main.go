package main

import (
	"bitcoinOrder/internal/app/orderchecker/service"
	"bitcoinOrder/internal/repository"
	"bitcoinOrder/pkg/database"
	"log"
	"time"
)

func main() {
	dbConfig := &database.Config{
		Host:     "localhost",
		User:     "postgres",
		Password: "postgres",
		DBName:   "order_app",
		Port:     "6432",
		SSLMode:  "disable",
		TimeZone: "UTC",
	}

	sqlDB, gormDB, err := database.NewDBConnection(dbConfig)
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	defer sqlDB.Close()
	defer func() {
		sqlDB, _ := gormDB.DB()
		sqlDB.Close()
	}()

	transactionRepo := repository.NewTransactionRepository(sqlDB)
	lockRepo := repository.NewLockRepository(sqlDB)
	transactionService := service.NewOrderCheckerService(transactionRepo, lockRepo, sqlDB)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		err := transactionService.ProcessTransactions()
		if err != nil {
			log.Printf("Error processing transactions: %v", err)
		}
	}
}
