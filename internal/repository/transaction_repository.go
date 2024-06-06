package repository

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/pkg/utils"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ITransactionRepository interface {
	FindSellOrders(ctx context.Context) ([]entity.Order, error)
	FindBuyOrders(ctx context.Context) ([]entity.Order, error)
	SaveMatches(ctx context.Context, orderMatches []entity.OrderMatch) error
	UpdateBalance(ctx context.Context, users []*entity.Users) error
	FindOrderById(ctx context.Context, orderId uuid.UUID) (entity.Order, error)
	FindUserById(ctx context.Context, userId uuid.UUID) (*entity.Users, error)
	SoftDeleteOrder(ctx context.Context, orderId uuid.UUID) error
	UpdateOrders(ctx context.Context, orders []*entity.Order) error
	SoftDeleteMatch(ctx context.Context, matchID uuid.UUID) error
	FetchMatch(ctx context.Context, orderID1 uuid.UUID, orderID2 uuid.UUID) (*entity.OrderMatch, error)
}

type TransactionRepository struct {
	db   *sql.DB
	gorm *gorm.DB
}

func NewTransactionRepository(gorm *gorm.DB, db *sql.DB) ITransactionRepository {
	return &TransactionRepository{
		gorm: gorm,
		db:   db,
	}
}

func (o *TransactionRepository) FindBuyOrders(ctx context.Context) ([]entity.Order, error) {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return nil, err
	}

	sqlStatement := `
        SELECT 
        o.id, o.user_id, o.type, o.order_quantity, o.order_price, o.order_status, o.created_at, o.completed_at,
        u.id AS user_id, u.email, u.btc_balance, u.usdt_balance, u.created_at AS user_created_at,
        u.updated_at AS user_updated_at, u.deleted_at AS user_deleted_at
    FROM orders o
    JOIN users u ON o.user_id = u.id
    WHERE o.order_status = true AND o.type = 'buy' AND o.deleted_at IS NULL
    ORDER BY o.order_price ASC, o.created_at ASC;
    `

	return o.scanOrders(tx, sqlStatement)
}

func (o *TransactionRepository) FindSellOrders(ctx context.Context) ([]entity.Order, error) {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return nil, err
	}

	sqlStatement := `
 SELECT 
        o.id, o.user_id, o.type, o.order_quantity, o.order_price, o.order_status, o.created_at, o.completed_at,
        u.id AS user_id, u.email, u.btc_balance, u.usdt_balance, u.created_at AS user_created_at,
        u.updated_at AS user_updated_at, u.deleted_at AS user_deleted_at
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
		var userCreatedAt, userUpdatedAt, userDeletedAt sql.NullTime

		err = rows.Scan(
			&order.ID, &userIDStr, &order.Type, &order.OrderQuantity,
			&order.OrderPrice, &order.OrderStatus, &order.CreatedAt, &order.CompletedAt,
			&order.User.ID, &order.User.Email, &order.User.BtcBalance, &order.User.UsdtBalance,
			&userCreatedAt, &userUpdatedAt, &userDeletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error while scanning row: %w", err)
		}
		order.UserID, err = uuid.Parse(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("wrong user id format: %w", err)
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
}

func (o *TransactionRepository) SaveMatches(ctx context.Context, orderMatches []entity.OrderMatch) error {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return err
	}

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
			i*5+1, i*5+2, i*5+3, i*5+4, i*5+5)
		params = append(params, match.ID, match.OrderID1, match.OrderID2, match.OrderQuantity, match.MatchedAt)
	}

	_, err = tx.ExecContext(ctx, sqlStatement, params...)
	if err != nil {
		return fmt.Errorf("an error occurred while saving order matches: %w", err)
	}

	return nil
}

func (o *TransactionRepository) UpdateBalance(ctx context.Context, users []*entity.Users) error {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return err
	}

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
		usdtBalanceCases += fmt.Sprintf("WHEN '%v' THEN $%d::float ", user.ID, i*3+2)
		btcBalanceCases += fmt.Sprintf("WHEN '%v' THEN $%d::float ", user.ID, i*3+3)
		params = append(params, user.ID, *user.UsdtBalance, *user.BtcBalance)

		if i > 0 {
			userIDs += ", "
		}
		userIDs += fmt.Sprintf("$%d", i*3+1)
	}
	finalStatement := fmt.Sprintf(sqlStatement, usdtBalanceCases, btcBalanceCases, userIDs)
	_, err = tx.ExecContext(ctx, finalStatement, params...)
	if err != nil {
		return fmt.Errorf("an error occurred while updating user balances: %w", err)
	}

	return nil
}

func (o *TransactionRepository) FindOrderById(ctx context.Context, orderID uuid.UUID) (entity.Order, error) {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return entity.Order{}, err
	}

	sqlStatement := `
        SELECT 
            o.id, o.user_id, o.type, o.order_quantity, o.order_price, o.order_status, 
            o.created_at, o.completed_at,
            u.id AS user_id, u.email, u.btc_balance, u.usdt_balance, u.created_at AS user_created_at,
            u.updated_at AS user_updated_at, u.deleted_at AS user_deleted_at
        FROM orders o
        JOIN users u ON o.user_id = u.id
        WHERE o.id = $1 AND o.deleted_at IS NULL;
    `

	var order entity.Order
	var userIDStr string
	var userCreatedAt, userUpdatedAt, userDeletedAt sql.NullTime

	err = tx.QueryRowContext(ctx, sqlStatement, orderID).Scan(
		&order.ID,
		&userIDStr,
		&order.Type,
		&order.OrderQuantity,
		&order.OrderPrice,
		&order.OrderStatus,
		&order.CreatedAt,
		&order.CompletedAt,
		&order.User.ID, &order.User.Email, &order.User.BtcBalance, &order.User.UsdtBalance,
		&userCreatedAt, &userUpdatedAt, &userDeletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.Order{}, fmt.Errorf("order not found: %w", err)
		}
		return entity.Order{}, fmt.Errorf("an error occurred while finding the order by ID: %w", err)
	}

	// UserID'yi UUID'ye dönüştür
	order.UserID, err = uuid.Parse(userIDStr)
	if err != nil {
		return entity.Order{}, fmt.Errorf("wrong user id format: %w", err)
	}

	// User bilgilerini doldur
	order.User.CreatedAt = userCreatedAt.Time
	if userUpdatedAt.Valid {
		t := userUpdatedAt.Time
		order.User.UpdatedAt = &t
	}
	if userDeletedAt.Valid {
		t := userDeletedAt.Time
		order.User.DeletedAt = &t
	}

	return order, nil
}

func (o *TransactionRepository) FindUserById(ctx context.Context, userID uuid.UUID) (*entity.Users, error) {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return nil, err
	}

	sqlStatement := `
        SELECT id, usdt_balance, btc_balance
        FROM users
        WHERE id = $1;
    `

	var user entity.Users
	err = tx.QueryRowContext(ctx, sqlStatement, userID).Scan(
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

func (o *TransactionRepository) SoftDeleteOrder(ctx context.Context, orderID uuid.UUID) error {
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
		return fmt.Errorf("an error occurred while soft deleting the order: %w", err)
	}

	return nil
}
func (o *TransactionRepository) UpdateOrders(ctx context.Context, orders []*entity.Order) error {
	// tx, err := utils.TxFromContext(ctx)
	// if err != nil {
	//  return err
	// }

	for _, order := range orders {
		result := o.gorm.WithContext(ctx).Model(&entity.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
			"order_quantity": order.OrderQuantity,
			"order_status":   order.OrderStatus,
			"completed_at":   order.CompletedAt,
		})

		if result.Error != nil {
			return fmt.Errorf("failed to update order with ID %s: %w", order.ID, result.Error)
		}

		if result.RowsAffected == 0 {
			return fmt.Errorf("no orders updated with ID %s", order.ID)
		}
	}

	return nil
}

func (o *TransactionRepository) SoftDeleteMatch(ctx context.Context, matchID uuid.UUID) error {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return err
	}

	sqlStatement := `
        UPDATE order_matches
        SET deleted_at = NOW()
        WHERE id = $1;
    `
	_, err = tx.ExecContext(ctx, sqlStatement, matchID)
	if err != nil {
		return fmt.Errorf("an error occurred while soft deleting the match: %w", err)
	}
	return nil
}

func (o *TransactionRepository) FetchMatch(ctx context.Context, orderID1 uuid.UUID, orderID2 uuid.UUID) (*entity.OrderMatch, error) {
	tx, err := utils.TxFromContext(ctx)
	if err != nil {
		return nil, err
	}

	sqlStatement := `
        SELECT id, order_id1, order_id2, order_quantity, matched_at 
        FROM order_matches 
        WHERE order_id1 = $1 AND order_id2 = $2 AND deleted_at IS NULL;
    `

	var match entity.OrderMatch
	err = tx.QueryRowContext(ctx, sqlStatement, orderID1, orderID2).Scan(
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
