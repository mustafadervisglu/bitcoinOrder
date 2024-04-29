package main

import (
	"bitcoinOrder/config/postgresql"
	"bitcoinOrder/controller"
	"bitcoinOrder/entity"
	"bitcoinOrder/repository"
	"bitcoinOrder/service"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"time"
)

func main() {
	e := echo.New()
	config := postgresql.NewConfig("localhost", "postgres", "postgres", "order_app", "6432", "disable", "UTC")
	db := postgresql.OpenDB(config)

	err := db.AutoMigrate(&entity.Order{}, &entity.OrderMatch{}, &entity.Users{})
	if err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	orderRepository := repository.NewOrderRepository(db)
	orderService := service.NewBitcoinOrderService(orderRepository)
	orderController := controller.NewBitcoinOrderController(orderService)
	orderController.RegisterRoutes(e)

	go func() {
		if err := e.Start(":8080"); err != nil {
			log.Fatal("Echo server shutdown: ", err)
		}
	}()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		//fmt.Println("Checking for matches...")
		matches, err := orderRepository.CheckOrder()
		if err != nil {
			log.Errorf("Failed to check orders: %v", err)
			continue
		}
		if len(matches) > 0 {
			fmt.Printf("Matches found: %v\n", matches)
		} else {
			//fmt.Println("No matches found.")
		}
	}
}
