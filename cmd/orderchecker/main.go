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

	db, err := database.NewDBConnection(dbConfig)
	if err != nil {
		log.Fatalf("Veritabanına bağlanılamadı: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	transactionRepo := repository.NewTransactionRepository(db)
	transactionService := service.NewOrderCheckerService(transactionRepo, db)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		err := transactionService.ProcessTransactions()
		if err != nil {
			log.Printf("Error processing transactions: %v", err)
		}
	}
}
