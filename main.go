package main

import (
	"bitcoinOrder/config/postgresql"
	"bitcoinOrder/controller"
	"bitcoinOrder/entity"
	"bitcoinOrder/repository"
	"bitcoinOrder/service"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
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
	if err := e.Start(":8080"); err != nil {
		log.Fatal("Echo server shutdown: ", err)
	}
}
