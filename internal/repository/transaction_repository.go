package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log"
)

type ITransactionRepository interface {
	FindSellOrders(db *sql.Tx) ([]entity.Order, error)
	FindBuyOrders(db *sql.Tx) ([]entity.Order, error)
	SaveMatches(db *sql.Tx, orderMatches []entity.OrderMatch) error
	UpdateBalance(db *sql.Tx, users []*entity.Users) error
	FindOrderById(db *sql.Tx, orderId uuid.UUID) (entity.Order, error)
	FindUserById(db *sql.Tx, userId uuid.UUID) (*entity.Users, error)
	SoftDeleteOrder(db *sql.Tx, orderId uuid.UUID) error
	UpdateOrders(db *sql.Tx, orders []*entity.Order) error
	SoftDeleteMatch(db *sql.Tx, matchID uuid.UUID) error
	FetchMatch(db *sql.Tx, orderID1 uuid.UUID, orderID2 uuid.UUID) (*entity.OrderMatch, error)
}

type TransactionRepository struct {
	db *sql.DB
}

func NewTransactionRepository(db *sql.DB) ITransactionRepository {
	return &TransactionRepository{

		db: db,
	}
}

func (o *TransactionRepository) FindBuyOrders(tx *sql.Tx) ([]entity.Order, error) {
	sqlStatement := `
        SELECT o.id, o.user_id, o.type, o.order_quantity, o.order_price, o.order_status, o.created_at, o.completed_at,
               u.usdt_balance, u.btc_balance
        FROM orders o
        JOIN users u ON o.user_id = u.id
        WHERE o.order_status = true AND o.type = 'buy' AND o.deleted_at IS NULL
        ORDER BY o.order_price ASC, o.created_at ASC;
    `

	return o.scanOrders(tx, sqlStatement)
}

func (o *TransactionRepository) FindSellOrders(tx *sql.Tx) ([]entity.Order, error) {
	sqlStatement := `
        SELECT o.id, o.user_id, o.type, o.order_quantity, o.order_price, o.order_status, o.created_at, o.completed_at,
               u.usdt_balance, u.btc_balance
        FROM orders o
        JOIN users u ON o.user_id = u.id
        WHERE o.order_status = true AND o.type = 'sell' AND o.deleted_at IS NULL
        ORDER BY o.order_price DESC, o.created_at ASC;
    `

	return o.scanOrders(tx, sqlStatement)
}

func (o *TransactionRepository) scanOrders(tx *sql.Tx, sqlStatement string) ([]entity.Order, error) {
	rows, err := tx.QueryContext(context.Background(), sqlStatement)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while retrieving orders: %w", err)
	}
	defer rows.Close()

	var orders []entity.Order
	for rows.Next() {
		var order entity.Order
		var userIDStr string
		err := rows.Scan(
			&order.ID,
			&userIDStr,
			&order.Type,
			&order.OrderQuantity,
			&order.OrderPrice,
			&order.OrderStatus,
			&order.CreatedAt,
			&order.CompletedAt,
			&order.User.UsdtBalance,
			&order.User.BtcBalance,
		)
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

func (o *TransactionRepository) SaveMatches(tx *sql.Tx, orderMatches []entity.OrderMatch) error {
	if len(orderMatches) == 0 {
		return nil
	}

	sqlStatement := `
        INSERT INTO order_matches (id, order_id1, order_id2, order_quantity, matched_at) 
        VALUES 
    `
	var params []interface{}

	for i, match := range orderMatches {
		if i > 0 {
			sqlStatement += ","
		}
		sqlStatement += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)",
			i*5+1, i*5+2, i*5+3, i*5+4, i*5+5) // Parametre yer tutucuları kullan

		params = append(params, match.ID, match.OrderID1, match.OrderID2, match.OrderQuantity, match.MatchedAt)
	}

	_, err := tx.ExecContext(context.Background(), sqlStatement, params...) // Parametreleri ilet
	if err != nil {
		return fmt.Errorf("an error occurred while saving order matches: %w", err)
	}

	return nil
}

func (o *TransactionRepository) UpdateBalance(tx *sql.Tx, users []*entity.Users) error {
	sqlStatement := `
        UPDATE users
        SET usdt_balance = CASE id 
                              %s
                          END,
            btc_balance = CASE id 
                              %s
                          END
        WHERE id IN (%s);
    `

	usdtBalanceCases := ""
	btcBalanceCases := ""
	userIDs := ""
	var params []interface{}

	for i, user := range users {
		usdtBalanceCases += fmt.Sprintf("WHEN $%d THEN $%d ", i*2+1, i*2+2) // Parametre yer tutucuları
		btcBalanceCases += fmt.Sprintf("WHEN $%d THEN $%d ", i*2+1, i*2+3)
		params = append(params, user.ID, *user.UsdtBalance, *user.BtcBalance) // Parametreleri ekle

		if i > 0 {
			userIDs += ", "
		}
		userIDs += fmt.Sprintf("$%d", i*2+1)
	}

	finalStatement := fmt.Sprintf(sqlStatement, usdtBalanceCases, btcBalanceCases, userIDs)
	_, err := tx.ExecContext(context.Background(), finalStatement, params...) // Parametreleri ilet
	if err != nil {
		return fmt.Errorf("an error occurred while updating user balances: %w", err)
	}

	return nil
}

func (o *TransactionRepository) FindOrderById(tx *sql.Tx, orderID uuid.UUID) (entity.Order, error) {
	sqlStatement := `
        SELECT o.id, o.user_id, o.type, o.order_quantity, o.order_price, o.order_status, o.created_at, o.completed_at,
               u.usdt_balance, u.btc_balance
        FROM orders o
        JOIN users u ON o.user_id = u.id
        WHERE o.id = $1 AND o.deleted_at IS NULL;
    `

	var order entity.Order
	var userIDStr string
	err := tx.QueryRowContext(context.Background(), sqlStatement, orderID).Scan(
		&order.ID,
		&userIDStr,
		&order.Type,
		&order.OrderQuantity,
		&order.OrderPrice,
		&order.OrderStatus,
		&order.CreatedAt,
		&order.CompletedAt,
		&order.User.UsdtBalance,
		&order.User.BtcBalance,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Order{}, fmt.Errorf("order not found: %w", err)
		}
		return entity.Order{}, fmt.Errorf("an error occurred while finding the order by ID: %w", err)
	}
	order.UserID, err = uuid.Parse(userIDStr)
	if err != nil {
		return entity.Order{}, fmt.Errorf("wrong user id format: %w", err)
	}
	return order, nil
}

func (o *TransactionRepository) FindUserById(tx *sql.Tx, userID uuid.UUID) (*entity.Users, error) {
	sqlStatement := `
        SELECT id, usdt_balance, btc_balance
        FROM users
        WHERE id = $1;
    `

	var user entity.Users
	err := tx.QueryRowContext(context.Background(), sqlStatement, userID).Scan(
		&user.ID,
		&user.UsdtBalance,
		&user.BtcBalance,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("an error occurred while finding the user by ID: %w", err)
	}

	return &user, nil
}

func (o *TransactionRepository) SoftDeleteOrder(tx *sql.Tx, orderID uuid.UUID) error {
	sqlStatement := `
        UPDATE orders
        SET deleted_at = NOW()
        WHERE id = $1;
    `

	_, err := tx.ExecContext(context.Background(), sqlStatement, orderID)
	if err != nil {
		return fmt.Errorf("an error occurred while soft deleting the order: %w", err)
	}

	return nil
}

func (o *TransactionRepository) UpdateOrders(tx *sql.Tx, orders []*entity.Order) error {
	sqlStatement := `
    UPDATE orders 
    SET order_quantity = CASE id %s END, 
        order_status = CASE id %s END, 
        completed_at = CASE id %s END 
    WHERE id IN (%s);
    `
	log.Println("test UpdateOrders")
	var params []interface{}
	var orderQuantityCases, orderStatusCases, completedAtCases, orderIDs string

	for i, order := range orders {
		orderQuantityCases += fmt.Sprintf("WHEN '%s' THEN $%d::float ", order.ID, i*3+1)
		orderStatusCases += fmt.Sprintf("WHEN '%s' THEN $%d::boolean ", order.ID, i*3+2)
		if order.CompletedAt != nil {
			completedAtCases += fmt.Sprintf("WHEN '%s' THEN to_timestamp($%d / 1000.0) ", order.ID, i*3+3)
			params = append(params, order.OrderQuantity, order.OrderStatus, order.CompletedAt.UnixNano()/1000.0)
		} else {
			completedAtCases += fmt.Sprintf("WHEN '%s' THEN NULL ", order.ID)
			params = append(params, order.OrderQuantity, order.OrderStatus)
		}

		if i > 0 {
			orderIDs += ", "
		}
		orderIDs += fmt.Sprintf("'%s'", order.ID)
	}
	log.Println("test1 UpdateOrders")

	finalStatement := fmt.Sprintf(sqlStatement, orderQuantityCases, orderStatusCases, completedAtCases, orderIDs)
	log.Println("test2 UpdateOrders")

	_, err := tx.ExecContext(context.Background(), finalStatement, params...)
	if err != nil {
		return fmt.Errorf("an error occurred while updating orders: %w", err)
	}
	return nil
}

func (o *TransactionRepository) SoftDeleteMatch(tx *sql.Tx, matchID uuid.UUID) error {
	sqlStatement := `
        UPDATE order_matches
        SET deleted_at = NOW()
        WHERE id = $1;
    `
	_, err := tx.ExecContext(context.Background(), sqlStatement, matchID)
	if err != nil {
		return fmt.Errorf("an error occurred while soft deleting the match: %w", err)
	}
	return nil
}

func (o *TransactionRepository) FetchMatch(tx *sql.Tx, orderID1 uuid.UUID, orderID2 uuid.UUID) (*entity.OrderMatch, error) {
	sqlStatement := `
        SELECT id, order_id1, order_id2, order_quantity, matched_at 
        FROM order_matches 
        WHERE order_id1 = $1 AND order_id2 = $2 AND deleted_at IS NULL;
    `

	var match entity.OrderMatch
	err := tx.QueryRowContext(context.Background(), sqlStatement, orderID1, orderID2).Scan(
		&match.ID, &match.OrderID1, &match.OrderID2, &match.OrderQuantity, &match.MatchedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("eşleşme getirilirken hata oluştu: %w", err)
	}

	return &match, nil
}
