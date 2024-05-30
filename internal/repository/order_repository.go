package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type IOrderRepository interface {
	FindOpenSellOrders() ([]entity.Order, error)
	FindOpenBuyOrders() ([]entity.Order, error)
	CreateOrder(tx *sql.DB, newOrder entity.Order) (entity.Order, error)
	SoftDeleteOrder(orderId string) error
	FindOpenOrdersByUser(tx *sql.DB, userID string) ([]entity.Order, error)
	FindAllOrders(tx *sql.DB) ([]entity.Order, error)
	UpdateOrder(tx *sql.DB, order entity.Order) error
}

type OrderRepository struct {
	gormDB *gorm.DB
	db     *sql.DB
}

func NewOrderRepository(gormDB *gorm.DB, db *sql.DB) *OrderRepository {
	return &OrderRepository{
		gormDB: gormDB,
		db:     db,
	}
}

func (o *OrderRepository) CreateOrder(tx *sql.DB, newOrder entity.Order) (entity.Order, error) {
	userIDStr := newOrder.UserID.String()
	var sqlStatement = `
        INSERT INTO orders (id, user_id, type, order_quantity, order_price, order_status, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, created_at;
    `
	err := tx.QueryRowContext(context.Background(), sqlStatement, newOrder.ID, userIDStr, newOrder.Type, newOrder.OrderQuantity, newOrder.OrderPrice, newOrder.OrderStatus, time.Now()).
		Scan(&newOrder.ID, &newOrder.CreatedAt)
	if err != nil {
		return entity.Order{}, fmt.Errorf("an error occurred while creating the order: %w", err)
	}
	return newOrder, nil
}

func (o *OrderRepository) UpdateOrder(tx *sql.DB, order entity.Order) error {
	sqlStatement := `
        UPDATE orders
        SET order_quantity = $1
        WHERE id = $2;
    `
	_, err := tx.ExecContext(context.Background(), sqlStatement, order.OrderQuantity, order.ID)
	if err != nil {
		return fmt.Errorf("an error occurred while updating the order: %w", err)
	}
	return nil
}

func (o *OrderRepository) SoftDeleteOrder(orderId string) error {
	if err := o.gormDB.Model(&entity.Order{}).Where("id = ?", orderId).Update("deleted_at", time.Now()).Error; err != nil {
		return err
	}
	return nil
}

func (o *OrderRepository) FindOpenSellOrders() ([]entity.Order, error) {
	var sellOrders []entity.Order
	if err := o.gormDB.Where("deleted_at IS NULL AND type = ?", "sell").
		Order("created_at ASC").
		Find(&sellOrders).Error; err != nil {
		return nil, err
	}
	return sellOrders, nil
}

func (o *OrderRepository) FindOpenBuyOrders() ([]entity.Order, error) {
	var buyOrders []entity.Order
	if err := o.gormDB.Where("deleted_at IS NULL AND type = ? ", "buy").
		Order("created_at DESC").
		Find(&buyOrders).Error; err != nil {
		return nil, err
	}
	return buyOrders, nil
}

func (o *OrderRepository) FindOpenOrdersByUser(tx *sql.DB, userID string) ([]entity.Order, error) {
	sqlStatement := `

        SELECT id, user_id, type, order_quantity, order_price, order_status, created_at, completed_at 
        FROM orders
        WHERE user_id = $1 AND deleted_at IS NULL AND order_status = true; 
    `

	rows, err := tx.QueryContext(context.Background(), sqlStatement, userID)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while retrieving orders: %w", err)
	}
	defer rows.Close()

	var orders []entity.Order
	for rows.Next() {
		var order entity.Order
		var userIDStr string
		err := rows.Scan(&order.ID, &userIDStr, &order.Type, &order.OrderQuantity, &order.OrderPrice, &order.OrderStatus, &order.CreatedAt, &order.CompletedAt)
		if err != nil {
			return nil, fmt.Errorf("error while scanning row: %w", err)
		}
		order.UserID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("wrong user id format: %w", err)
		}

		orders = append(orders, order)
	}

	return orders, nil
}

func (o *OrderRepository) FindAllOrders(tx *sql.DB) ([]entity.Order, error) {
	sqlStatement := `
        SELECT id, user_id, type, order_quantity, order_price, order_status, created_at, completed_at 
        FROM orders
        WHERE deleted_at IS NULL;
    `

	rows, err := tx.QueryContext(context.Background(), sqlStatement)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while receiving orders: %w", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			return
		}
	}(rows)

	var orders []entity.Order
	for rows.Next() {
		var order entity.Order
		var userIDStr string
		err := rows.Scan(&order.ID, &userIDStr, &order.Type, &order.OrderQuantity, &order.OrderPrice, &order.OrderStatus, &order.CreatedAt, &order.CompletedAt)
		if err != nil {
			return nil, fmt.Errorf("error while scanning row: %w", err)
		}
		order.UserID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("wrong user id format: %w", err)
		}

		orders = append(orders, order)
	}

	return orders, nil
}
