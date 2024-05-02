package utils

import "fmt"

type OrderType string

const (
	BuyOrder  = "buy"
	SellOrder = "sell"
)

func ValidateOrderType(orderType OrderType) error {
	switch orderType {
	case BuyOrder, SellOrder:
		return nil
	default:
		return fmt.Errorf("invalid order type: %s", orderType)
	}
}
