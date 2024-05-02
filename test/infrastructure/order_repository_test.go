package infrastructure

import (
	"bitcoinOrder/internal/domain/entity"
	"bitcoinOrder/pkg/config/postgresql"
	"bitcoinOrder/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"os"
	"testing"
)

var orderRepository repository.IOrderRepository
var db *gorm.DB

func TestMain(m *testing.M) {

	db = postgresql.OpenDB(postgresql.NewConfig(
		"localhost", "postgres", "postgres", "order_app", "6432", "disable", "UTC"))

	orderRepository = repository.NewOrderRepository(db)
	exitCode := m.Run()

	os.Exit(exitCode)
}

func clearData(db *gorm.DB) {
	TestDataClear(db)
}

func TestCreateUser(t *testing.T) {
	usdt := 1000.00
	btc := 0.00
	email := "mustafadervisoglu16@gmail.com"
	user := entity.Users{
		Email:      email,
		BtcBalance: &usdt,
		UsdBalance: &btc,
	}

	t.Run("Create User", func(t *testing.T) {
		actualOrder, err := orderRepository.CreateUser(user)
		assert.NoError(t, err, "Error while creating user")
		assert.NotEqual(t, uuid.Nil, actualOrder.ID, "Expected user ID to be set")
	})
	t.Run("Create User with existing email", func(t *testing.T) {
		_, err := orderRepository.CreateUser(user)
		assert.Error(t, err, "Expected error while creating user with existing email")
	})

}

func TestFindUserById(t *testing.T) {
	clearData(db)

	usdt := 1000.00
	btc := 0.00
	email := "testuser@example.com"
	user := entity.Users{
		Email:      email,
		BtcBalance: &usdt,
		UsdBalance: &btc,
	}
	createdUser, err := orderRepository.CreateUser(user)
	assert.NoError(t, err, "Error while creating user")
	assert.NotEqual(t, uuid.Nil, createdUser.ID, "Expected user ID to be set")

	t.Run("Find User By Valid ID", func(t *testing.T) {
		foundUser, err := orderRepository.GetBalance(createdUser.ID.String())
		assert.NoError(t, err, "Should find user without error")
		assert.Equal(t, createdUser.ID, foundUser.ID, "The IDs should match")
		assert.Equal(t, createdUser.Email, foundUser.Email, "The emails should match")
	})

	t.Run("User Not Found", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		_, err := orderRepository.GetBalance(nonExistentID)
		assert.Error(t, err, "Should not find user")
	})

	clearData(db)
}
