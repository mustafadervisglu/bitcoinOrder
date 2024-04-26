package dto

import "github.com/google/uuid"

type OrderDto struct {
	Asset         string
	OrderPrice    float64
	OrderQuantity float64
	OrderStatus   bool
}
type UserDto struct {
	Email      string  `json:"Email"`
	BtcBalance float64 `json:"BtcBalance"`
	UsdBalance float64 `json:"UsdBalance"`
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
