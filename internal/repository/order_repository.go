package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/pkg/utils"
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
	CreateOrder(ctx context.Context, newOrder entity.Order) (entity.Order, error)
	SoftDeleteOrder(ctx context.Context, orderId uuid.UUID) error
	FindOpenOrdersByUser(ctx context.Context, userID uuid.UUID) ([]entity.Order, error)
	FindAllOrders(ctx context.Context) ([]entity.Order, error)
	UpdateOrder(ctx context.Context, order entity.Order) error
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

func (o *OrderRepository) CreateOrder(ctx context.Context, newOrder entity.Order) (entity.Order, error) {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return entity.Order{}, err
	}

	userIDStr := newOrder.UserID.String()
	var sqlStatement = `
        INSERT INTO orders (id, user_id, type, order_quantity, order_price, order_status, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id, created_at;
    `
	err = tx.QueryRowContext(ctx, sqlStatement, newOrder.ID, userIDStr, newOrder.Type, newOrder.OrderQuantity, newOrder.OrderPrice, newOrder.OrderStatus, time.Now()).
		Scan(&newOrder.ID, &newOrder.CreatedAt)
	if err != nil {
		return entity.Order{}, fmt.Errorf("an error occurred while creating the order: %w", err)
	}

	return newOrder, nil
}

func (o *OrderRepository) UpdateOrder(ctx context.Context, order entity.Order) error {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return err
	}
	sqlStatement := `
        UPDATE orders
        SET order_quantity = $1
        WHERE id = $2;
    `
	_, err = tx.ExecContext(ctx, sqlStatement, order.OrderQuantity, order.ID)
	if err != nil {
		return fmt.Errorf("an error occurred while updating the order: %w", err)
	}
	return nil
}

func (o *OrderRepository) SoftDeleteOrder(ctx context.Context, orderID uuid.UUID) error {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return err
	}
	sqlStatement := `
        UPDATE orders
        SET deleted_at = NOW()
        WHERE id = $1;
    `
	_, err = tx.ExecContext(ctx, sqlStatement, orderID)
	if err != nil {
		return fmt.Errorf("failed to soft delete order with ID %s: %w", orderID, err)
	}
	return nil
}

func (o *OrderRepository) FindOpenSellOrders() ([]entity.Order, error) {
	sqlStatement := `
        SELECT id, user_id, type, order_quantity, order_price, order_status, created_at, completed_at 
        FROM orders
        WHERE deleted_at IS NULL AND type = 'sell'
        ORDER BY created_at ASC;
    `
	ctx := context.Background()
	return o.fetchOrders(ctx, sqlStatement)
}

func (o *OrderRepository) FindOpenBuyOrders() ([]entity.Order, error) {
	sqlStatement := `
        SELECT id, user_id, type, order_quantity, order_price, order_status, created_at, completed_at 
        FROM orders
        WHERE deleted_at IS NULL AND type = 'buy' 
        ORDER BY created_at DESC;
    `

	ctx := context.Background()
	return o.fetchOrders(ctx, sqlStatement)
}

func (o *OrderRepository) FindOpenOrdersByUser(ctx context.Context, userID uuid.UUID) ([]entity.Order, error) {
	sqlStatement := `
        SELECT id, user_id, type, order_quantity, order_price, order_status, created_at, completed_at
        FROM orders
        WHERE user_id = $1 AND deleted_at IS NULL AND order_status = true; 
    `
	return o.fetchOrdersByUser(ctx, sqlStatement, userID)
}

func (o *OrderRepository) FindAllOrders(ctx context.Context) ([]entity.Order, error) {
	sqlStatement := `
     SELECT
		 o.id, o.user_id, o.type, o.order_quantity, o.order_price, o.order_status,
		 o.created_at, o.completed_at,
		 u.id AS user_id, u.email, u.btc_balance, u.usdt_balance, u.created_at AS user_created_at,
		 u.updated_at AS user_updated_at, u.deleted_at AS user_deleted_at
	 FROM orders o 
     JOIN users u ON o.user_id = u.id
     WHERE o.deleted_at IS NULL;
    `
	return o.fetchOrdersWithUser(ctx, sqlStatement)
}

func (o *OrderRepository) fetchOrders(ctx context.Context, sqlStatement string, args ...interface{}) ([]entity.Order, error) {
	var rows *sql.Rows

	// Context'te transaction varsa kullan, yoksa *sql.DB kullan
	if tx, err := utils.TxFromContext(ctx); err == nil {
		rows, err = tx.QueryContext(ctx, sqlStatement, args...)
	} else {
		rows, err = o.db.QueryContext(ctx, sqlStatement, args...)
		if err != nil {
			return nil, fmt.Errorf("an error occurred while retrieving orders: %w", err)
		}
	}

	if rows != nil {
		defer rows.Close()

		var orders []entity.Order
		for rows.Next() {
			var order entity.Order
			var userIDStr string

			var userEmail sql.NullString
			var userBtcBalance, userUsdtBalance sql.NullFloat64
			var userCreatedAt, userUpdatedAt, userDeletedAt sql.NullTime

			err := rows.Scan(
				&order.ID, &userIDStr, &order.Type, &order.OrderQuantity,
				&order.OrderPrice, &order.OrderStatus, &order.CreatedAt, &order.CompletedAt,
				&order.User.ID, &userEmail, &userBtcBalance, &userUsdtBalance,
				&userCreatedAt, &userUpdatedAt, &userDeletedAt,
			)
			if err != nil {
				return nil, fmt.Errorf("error while scanning row: %w", err)
			}

			order.UserID, err = uuid.Parse(userIDStr)
			if err != nil {
				return nil, fmt.Errorf("wrong user id format: %w", err)
			}

			order.User.Email = userEmail.String
			if userBtcBalance.Valid {
				order.User.BtcBalance = &userBtcBalance.Float64
			}
			if userUsdtBalance.Valid {
				order.User.UsdtBalance = &userUsdtBalance.Float64
			}
			order.User.CreatedAt = userCreatedAt.Time
			if userUpdatedAt.Valid {
				t := userUpdatedAt.Time
				order.User.UpdatedAt = &t
			}
			if userDeletedAt.Valid {
				t := userDeletedAt.Time
				order.User.DeletedAt = &t
			}

			orders = append(orders, order)
		}

		return orders, nil
	} else {
		return nil, fmt.Errorf("rows nil")
	}
}

func (o *OrderRepository) fetchOrdersWithUser(ctx context.Context, sqlStatement string, args ...interface{}) ([]entity.Order, error) {
	var rows *sql.Rows

	if tx, err := utils.TxFromContext(ctx); err == nil {
		rows, err = tx.QueryContext(ctx, sqlStatement, args...)
	} else {
		rows, err = o.db.QueryContext(ctx, sqlStatement, args...)
		if err != nil {
			return nil, fmt.Errorf("an error occurred while retrieving orders: %w", err)
		}
	}

	if rows != nil {
		defer rows.Close()

		var orders []entity.Order
		for rows.Next() {
			var order entity.Order
			var userIDStr string

			var userEmail sql.NullString
			var userBtcBalance, userUsdtBalance sql.NullFloat64
			var userCreatedAt, userUpdatedAt, userDeletedAt sql.NullTime

			err := rows.Scan(
				&order.ID, &userIDStr, &order.Type, &order.OrderQuantity,
				&order.OrderPrice, &order.OrderStatus, &order.CreatedAt, &order.CompletedAt,
				&order.User.ID, &userEmail, &userBtcBalance, &userUsdtBalance,
				&userCreatedAt, &userUpdatedAt, &userDeletedAt,
			)
			if err != nil {
				return nil, fmt.Errorf("error while scanning row: %w", err)
			}

			order.UserID, err = uuid.Parse(userIDStr)
			if err != nil {
				return nil, fmt.Errorf("wrong user id format: %w", err)
			}

			order.User.Email = userEmail.String
			if userBtcBalance.Valid {
				order.User.BtcBalance = &userBtcBalance.Float64
			}
			if userUsdtBalance.Valid {
				order.User.UsdtBalance = &userUsdtBalance.Float64
			}
			order.User.CreatedAt = userCreatedAt.Time
			if userUpdatedAt.Valid {
				t := userUpdatedAt.Time
				order.User.UpdatedAt = &t
			}
			if userDeletedAt.Valid {
				t := userDeletedAt.Time
				order.User.DeletedAt = &t
			}

			orders = append(orders, order)
		}

		return orders, nil
	} else {
		return nil, fmt.Errorf("rows nil")
	}
}

func (o *OrderRepository) fetchOrdersByUser(ctx context.Context, sqlStatement string, args ...interface{}) ([]entity.Order, error) {
	var rows *sql.Rows

	if tx, err := utils.TxFromContext(ctx); err == nil {
		rows, err = tx.QueryContext(ctx, sqlStatement, args...)
	} else {
		rows, err = o.db.QueryContext(ctx, sqlStatement, args...)
		if err != nil {
			return nil, fmt.Errorf("an error occurred while retrieving orders: %w", err)
		}
	}

	if rows != nil {
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
	} else {
		return nil, fmt.Errorf("rows nil")
	}
}
