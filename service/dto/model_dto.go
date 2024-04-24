package dto

import "github.com/google/uuid"

type OrderDto struct {
	Asset         string
	OrderPrice    float64
	OrderQuantity float64
	OrderStatus   bool
}
type UserDto struct {
	Email      string
	BtcBalance float64
	UsdBalance float64
}

type OrderMatchDto struct {
	OrderID1 uuid.UUID
	OrderID2 uuid.UUID
}

type BalanceDto struct {
	Id     string
	Asset  string
	Amount float64
}
