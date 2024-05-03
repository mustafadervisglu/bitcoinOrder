package main

import (
	"bitcoinOrder/internal/app/orderchecker/service"
	"bitcoinOrder/internal/repository"
	"bitcoinOrder/pkg/config/postgresql"
	"fmt"
	"log"
	"time"
)

func main() {
	config := postgresql.NewConfig("localhost", "postgres", "postgres", "order_app", "6432", "disable", "UTC")
	db := postgresql.OpenDB(config)

	transactionRepo := repository.NewTransactionRepository(db)
	transactionService := service.NewOrderCheckerService(transactionRepo)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		err := transactionService.ProcessTransactions()
		if err != nil {
			log.Printf("Error processing transactions: %v", err)
		} else {
			fmt.Println("Transactions processed successfully.")
		}
	}
}
