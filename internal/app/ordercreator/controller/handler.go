package controller

import (
	"bitcoinOrder/internal/app/ordercreator/service"
	"bitcoinOrder/internal/common/dto"
	"github.com/labstack/echo/v4"
	"net/http"
)

type Handler struct {
	Service service.IOrderCreatorService
}

func NewOrderCreatorHandler(service service.IOrderCreatorService) *Handler {
	return &Handler{Service: service}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	e.POST("/api/v1/order", h.CreateOrder)
	e.POST("/api/v1/user", h.CreateUser)
	e.POST("/api/v1/user/addBalance/:id/:asset", h.AddBalance)
	e.GET("/api/v1/user/:id", h.GetBalance)
	e.GET("/api/v1/allOrder", h.FindAllOrder)
}

func (h *Handler) CreateOrder(c echo.Context) error {
	var orderDTO dto.OrderDto
	if err := c.Bind(&orderDTO); err != nil {
		return c.JSON(http.StatusBadRequest, "Invalid request data")
	}

	//TODO: this is optional, but it's a good practice to validate the request data
	if orderDTO.OrderPrice <= 0 || orderDTO.OrderQuantity <= 0 {
		return c.JSON(http.StatusBadRequest, "Order price and quantity must be positive")
	}

	createdOrder, err := h.Service.CreateOrder(orderDTO)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Failed to create order")
	}

	return c.JSON(http.StatusCreated, createdOrder)
}

func (h *Handler) CreateUser(e echo.Context) error {
	userDto := new(dto.UserDto)
	if err := e.Bind(userDto); err != nil {
		return e.JSON(http.StatusBadRequest, err.Error())
	}

	createdUser, err := h.Service.CreateUser(*userDto)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, err.Error())
	}
	return e.JSON(http.StatusCreated, createdUser)
}
func (h *Handler) AddBalance(e echo.Context) error {
	var balance dto.BalanceDto

	balance.Id = e.Param("id")
	balance.Asset = e.Param("asset")

	if err := e.Bind(&balance); err != nil {
		return e.JSON(http.StatusBadRequest, err.Error())
	}

	if err := h.Service.AddBalance(balance); err != nil {
		return e.JSON(http.StatusInternalServerError, err.Error())
	}
	return e.JSON(http.StatusOK, "balance updated successfully")
}
func (h *Handler) GetBalance(e echo.Context) error {
	id := e.Param("id")
	balance, err := h.Service.GetBalance(id)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, err.Error())
	}
	return e.JSON(http.StatusOK, balance)
}

func (h *Handler) FindAllOrder(e echo.Context) error {
	orders, err := h.Service.FindAllOrder()
	if err != nil {
		return e.JSON(http.StatusInternalServerError, err.Error())
	}
	return e.JSON(http.StatusOK, orders)
}
