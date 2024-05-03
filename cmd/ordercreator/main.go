package main

import (
	"bitcoinOrder/internal/app/ordercreator/controller"
	"bitcoinOrder/internal/app/ordercreator/service"
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/internal/repository"
	"bitcoinOrder/pkg/config/postgresql"
	"github.com/labstack/echo/v4"
	"log"
)

func main() {
	e := echo.New()
	config := postgresql.NewConfig("localhost", "postgres", "postgres", "order_app", "6432", "disable", "UTC")
	db := postgresql.OpenDB(config)

	err := db.AutoMigrate(&entity.Order{}, &entity.OrderMatch{}, &entity.Users{})
	if err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	orderRepo := repository.NewOrderRepository(db)
	userRepo := repository.NewUserRepository(db)
	orderService := service.NewOrderCreatorService(orderRepo, userRepo)
	orderHandler := controller.NewOrderCreatorHandler(orderService)

	orderHandler.RegisterRoutes(e)

	log.Fatal(e.Start(":8080"))
}
