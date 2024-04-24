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
	e.GET("/api/v1/user/:id", b.FindUserByID)

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

func (b *BitcoinOrderController) FindUserByID(c echo.Context) error {
	id := c.Param("id")
	user, err := b.bitcoinOrderService.FindUserByID(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, "Bad Request")
	}
	return c.JSON(http.StatusOK, user)
}
