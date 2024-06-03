package controller

import (
	"bitcoinOrder/internal/app/ordercreator/service"
	"bitcoinOrder/internal/common/dto"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
)

type Handler struct {
	Service service.IOrderCreatorService
}

func NewOrderCreatorHandler(service *service.OrderCreatorService) *Handler {
	return &Handler{Service: service}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	e.POST("/api/v1/order", h.CreateOrder)
	e.POST("/api/v1/user", h.CreateUser)
	e.POST("/api/v1/user/addBalance/:id/:asset", h.AddBalance)
	e.GET("/api/v1/user/:id", h.GetBalance)
	e.GET("/api/v1/allOrder", h.FindAllOrder)
	e.GET("api/v1/allUser", h.FindAllUser)
	e.GET("api/v1/findUser/:id", h.FindUser)
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

	err := h.Service.CreateOrder(orderDTO)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return nil
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
	var err error
	balance.Id, err = uuid.Parse(e.Param("id"))
	if err != nil {
		return e.JSON(http.StatusBadRequest, "Invalid user ID")
	}
	balance.Asset = e.Param("asset")

	if err := e.Bind(&balance); err != nil {
		return e.JSON(http.StatusBadRequest, err.Error())
	}

	ctx := e.Request().Context()
	if err := h.Service.AddBalance(ctx, balance); err != nil {
		return e.JSON(http.StatusInternalServerError, err.Error())
	}
	return e.JSON(http.StatusOK, "Balance updated successfully")
}

func (h *Handler) GetBalance(e echo.Context) error {
	id, err := uuid.Parse(e.Param("id"))
	if err != nil {
		return e.JSON(http.StatusBadRequest, "Invalid user ID")
	}

	balance, err := h.Service.GetBalance(id)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, err.Error())
	}
	return e.JSON(http.StatusOK, balance)
}

func (h *Handler) FindAllOrder(e echo.Context) error {
	ctx := e.Request().Context()
	orders, err := h.Service.FindAllOrder(ctx)
	if err != nil {
		return e.JSON(http.StatusInternalServerError, err.Error())
	}
	return e.JSON(http.StatusOK, orders)
}

func (h *Handler) FindAllUser(c echo.Context) error {
	users, err := h.Service.FindAllUser()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, users)
}

func (h *Handler) FindUser(e echo.Context) error {
	ctx := e.Request().Context()
	id, errID := uuid.Parse(e.Param("id"))
	if errID != nil {
		return e.JSON(http.StatusBadRequest, "Invalid user ID")
	}
	user, err := h.Service.FindUser(ctx, id)

	if err != nil {
		return e.JSON(http.StatusInternalServerError, err.Error())
	}
	return e.JSON(http.StatusOK, user)
}
