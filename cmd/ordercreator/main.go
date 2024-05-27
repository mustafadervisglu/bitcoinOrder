package main

import (
	"bitcoinOrder/internal/app/ordercreator/controller"
	"bitcoinOrder/internal/app/ordercreator/service"
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/internal/repository"
	"bitcoinOrder/pkg/database"
	"github.com/labstack/echo/v4"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func main() {

	e := echo.New()

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
		log.Fatalf("could not connect to database: %v", err)
	}
	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	err = db.AutoMigrate(&entity.Order{}, &entity.OrderMatch{}, &entity.Users{})
	if err != nil {
		log.Fatalf("An error occurred while creating tables: %v", err)
	}

	orderRepo := repository.NewOrderRepository(db)
	userRepo := repository.NewUserRepository(db)
	orderService := service.NewOrderCreatorService(orderRepo, userRepo, db)
	orderHandler := controller.NewOrderCreatorHandler(orderService)
	orderHandler.RegisterRoutes(e)
	log.Fatal(e.Start(":8080"))

}
