package controller

import (
	"bitcoinOrder/entity"
	"bitcoinOrder/service"
	"bitcoinOrder/service/dto"
	"github.com/labstack/echo/v4"
	"net/http"
)

type BitcoinOrderController struct {
	bitcoinOrderService service.IBitcoinOrderService
}

func NewBitcoinOrderController(bitcoinOrderService service.IBitcoinOrderService) *BitcoinOrderController {
	return &BitcoinOrderController{bitcoinOrderService: bitcoinOrderService}
}
func (b *BitcoinOrderController) RegisterRoutes(e *echo.Echo) {

	e.POST("/api/v1/user", b.CreateUser)
	e.POST("/api/v1/user/addBalance/:id/:asset", b.AddBalance)
	e.POST("/api/v1/order", b.CreateOrder)
	e.GET("/api/v1/user/:id", b.GetBalance)
	e.GET("/api/v1/allOrder", b.FindAllOrder)

}

func (b *BitcoinOrderController) CreateUser(c echo.Context) error {
	user := new(entity.Users)
	if err := c.Bind(user); err != nil {
		return c.JSON(http.StatusBadRequest, "Bad Request")
	}
	userDto := dto.UserDto{
		Email:      user.Email,
		UsdBalance: 0,
		BtcBalance: 0,
	}
	if user.UsdBalance != nil {
		userDto.UsdBalance = *user.UsdBalance
	}
	if user.BtcBalance != nil {
		userDto.BtcBalance = *user.BtcBalance
	}
	err2 := b.bitcoinOrderService.CreateUser(userDto)
	if err2 != nil {
		return c.JSON(http.StatusBadRequest, "Bad Request")
	}
	return c.JSON(http.StatusOK, "User created successfully")
}

func (b *BitcoinOrderController) GetBalance(c echo.Context) error {
	id := c.Param("id")
	user, err := b.bitcoinOrderService.GetBalance(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Bad Request")
	}
	return c.JSON(http.StatusOK, user)
}

func (b *BitcoinOrderController) AddBalance(c echo.Context) error {
	id := c.Param("id")
	asset := c.Param("asset")

	balance := new(dto.BalanceDto)
	if err := c.Bind(balance); err != nil {
		return c.JSON(http.StatusBadRequest, "Bad Request")
	}

	balance.Id = id
	balance.Asset = asset

	if err := b.bitcoinOrderService.AddBalance(*balance); err != nil {
		return c.JSON(http.StatusBadRequest, "Bad Request")
	}
	return c.JSON(http.StatusOK, "Balance added successfully")
}

func (b *BitcoinOrderController) FindAllOrder(c echo.Context) error {
	orders, _ := b.bitcoinOrderService.FindAllOrder()
	return c.JSON(http.StatusOK, orders)
}

func (b *BitcoinOrderController) CreateOrder(c echo.Context) error {
	order := new(dto.OrderDto)
	if err := c.Bind(order); err != nil {
		return c.JSON(http.StatusBadRequest, "Bad Request")
	}
	err := b.bitcoinOrderService.CreateOrder(*order)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Bad Request")
	}
	return c.JSON(http.StatusOK, "Order created successfully")
}
