package dto

import "github.com/google/uuid"

type OrderDto struct {
	Asset         string  `json:"Asset"`
	OrderPrice    float64 `json:"OrderPrice"`
	OrderQuantity float64 `json:"OrderQuantity"`
	OrderStatus   bool    `json:"OrderStatus"`
	UserID        string  `json:"UserID"`
	Type          string  `json:"Type"`
}
type UserDto struct {
	Email       string  `json:"Email"`
	BtcBalance  float64 `json:"BtcBalance"`
	UsdtBalance float64 `json:"UsdtBalance"`
}

type OrderMatchDto struct {
	OrderID1 uuid.UUID
	OrderID2 uuid.UUID
}

type BalanceDto struct {
	Id     string  `param:"Id"`
	Asset  string  `param:"Asset"`
	Amount float64 `json:"Amount"`
}
