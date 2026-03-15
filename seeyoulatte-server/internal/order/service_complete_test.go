package order

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/darkphotonKN/seeyoulatte-app/internal/constants"
	"github.com/darkphotonKN/seeyoulatte-app/internal/listing"
	dbutils "github.com/darkphotonKN/seeyoulatte-app/internal/utils/db"
	"github.com/darkphotonKN/seeyoulatte-app/internal/utils/errorutils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MockRepository implements the Repository interface for testing
type MockRepository struct {
	CreateFunc   func(ctx context.Context, order *Order) error
	CreateTxFunc func(ctx context.Context, tx *sqlx.Tx, order *Order) error
	GetAllFunc   func(ctx context.Context) ([]Order, error)
	UpdateFunc   func(ctx context.Context, order *Order) error
	DeleteFunc   func(ctx context.Context, id uuid.UUID) error
}

func (m *MockRepository) Create(ctx context.Context, order *Order) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, order)
	}
	return nil
}

func (m *MockRepository) CreateTx(ctx context.Context, tx *sqlx.Tx, order *Order) error {
	if m.CreateTxFunc != nil {
		return m.CreateTxFunc(ctx, tx, order)
	}
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	return nil
}

func (m *MockRepository) GetAll(ctx context.Context) ([]Order, error) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(ctx)
	}
	return []Order{}, nil
}

func (m *MockRepository) Update(ctx context.Context, order *Order) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, order)
	}
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

// MockListingService implements the ListingService interface for testing
type MockListingService struct {
	GetByIDWithSellerForUpdateTxFunc func(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) (*listing.ListingWithSeller, error)
	UpdateTxFunc                     func(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, sellerID uuid.UUID, req *listing.UpdateListingRequest) (*listing.Listing, error)
}

func (m *MockListingService) GetByIDWithSellerForUpdateTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) (*listing.ListingWithSeller, error) {
	if m.GetByIDWithSellerForUpdateTxFunc != nil {
		return m.GetByIDWithSellerForUpdateTxFunc(ctx, tx, id)
	}
	return &listing.ListingWithSeller{}, nil
}

func (m *MockListingService) UpdateTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, sellerID uuid.UUID, req *listing.UpdateListingRequest) (*listing.Listing, error) {
	if m.UpdateTxFunc != nil {
		return m.UpdateTxFunc(ctx, tx, id, sellerID, req)
	}
	return &listing.Listing{}, nil
}

// MockUserService implements the UserService interface for testing
type MockUserService struct {
	VerifyUserNotFrozenFunc func(ctx context.Context, id uuid.UUID) error
}

func (m *MockUserService) VerifyUserNotFrozen(ctx context.Context, id uuid.UUID) error {
	if m.VerifyUserNotFrozenFunc != nil {
		return m.VerifyUserNotFrozenFunc(ctx, id)
	}
	return nil
}

// TestCreateOrder_SuccessfulFlow demonstrates testing a successful order creation
// with proper transaction handling using sqlmock
func TestCreateOrder_SuccessfulFlow(t *testing.T) {
	// Setup test data
	buyerID := uuid.New()
	sellerID := uuid.New()
	listingID := uuid.New()
	orderID := uuid.New()

	// Create sqlmock database
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	// Setup mock expectations for transaction
	mock.ExpectBegin()
	mock.ExpectCommit()

	// Create mocks
	mockRepo := &MockRepository{
		CreateTxFunc: func(ctx context.Context, tx *sqlx.Tx, order *Order) error {
			// Verify order data
			if order.ListingID != listingID {
				t.Errorf("Expected listing ID %v, got %v", listingID, order.ListingID)
			}
			if order.BuyerID != buyerID {
				t.Errorf("Expected buyer ID %v, got %v", buyerID, order.BuyerID)
			}
			if order.SellerID != sellerID {
				t.Errorf("Expected seller ID %v, got %v", sellerID, order.SellerID)
			}
			if order.Quantity != 2 {
				t.Errorf("Expected quantity 2, got %d", order.Quantity)
			}
			if order.Amount != 51.00 {
				t.Errorf("Expected amount 51.00, got %f", order.Amount)
			}
			if order.State != string(constants.StatePendingPayment) {
				t.Errorf("Expected state %s, got %s", constants.StatePendingPayment, order.State)
			}

			// Simulate DB setting the ID
			order.ID = orderID
			return nil
		},
	}

	mockListingService := &MockListingService{
		GetByIDWithSellerForUpdateTxFunc: func(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) (*listing.ListingWithSeller, error) {
			return &listing.ListingWithSeller{
				ID:           listingID,
				SellerID:     sellerID,
				Title:        "Ethiopian Coffee",
				Price:        25.50,
				Quantity:     10,
				UserIsFrozen: false,
			}, nil
		},
		UpdateTxFunc: func(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, sID uuid.UUID, req *listing.UpdateListingRequest) (*listing.Listing, error) {
			// Verify quantity decrement
			if req.Quantity == nil || *req.Quantity != 8 {
				t.Errorf("Expected quantity to be decremented to 8, got %v", req.Quantity)
			}
			return &listing.Listing{}, nil
		},
	}

	mockUserService := &MockUserService{
		VerifyUserNotFrozenFunc: func(ctx context.Context, id uuid.UUID) error {
			if id != buyerID {
				t.Errorf("Expected buyer ID %v, got %v", buyerID, id)
			}
			return nil
		},
	}

	// Create service
	service := NewService(mockRepo, sqlxDB, slog.Default(), mockListingService, mockUserService)

	// Execute
	req := &CreateOrderRequest{
		ListingID: listingID,
		Quantity:  2,
	}

	order, err := service.Create(context.Background(), buyerID, req)

	// Verify results
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if order == nil {
		t.Fatal("Expected order, got nil")
	}
	if order.ID != orderID {
		t.Errorf("Expected order ID %v, got %v", orderID, order.ID)
	}

	// Verify all SQL expectations were met
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled sqlmock expectations: %v", err)
	}
}

// TestCreateOrder_BuyerFrozen tests the case where buyer is frozen
func TestCreateOrder_BuyerFrozen(t *testing.T) {
	buyerID := uuid.New()

	// No DB needed since we should fail before transaction
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create sqlmock: %v", err)
	}
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "sqlmock")

	// No transaction should be started
	// (no expectations set on mock)

	mockRepo := &MockRepository{}
	mockListingService := &MockListingService{}
	mockUserService := &MockUserService{
		VerifyUserNotFrozenFunc: func(ctx context.Context, id uuid.UUID) error {
			return errorutils.ErrUserIsFrozen
		},
	}

	service := NewService(mockRepo, sqlxDB, slog.Default(), mockListingService, mockUserService)

	req := &CreateOrderRequest{
		ListingID: uuid.New(),
		Quantity:  2,
	}

	order, err := service.Create(context.Background(), buyerID, req)

	// Verify error
	if !errors.Is(err, errorutils.ErrBuyerIsFrozen) {
		t.Errorf("Expected ErrBuyerIsFrozen, got %v", err)
	}
	if order != nil {
		t.Error("Expected nil order when buyer is frozen")
	}

	// Verify no SQL operations occurred
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("Unfulfilled sqlmock expectations: %v", err)
	}
}

// TestCreateOrder_TransactionRollback tests that the transaction is rolled back on errors
func TestCreateOrder_TransactionRollback(t *testing.T) {
	tests := []struct {
		name       string
		setupMocks func(*MockRepository, *MockListingService, *MockUserService)
		expectErr  bool
	}{
		{
			name: "rollback on seller frozen",
			setupMocks: func(repo *MockRepository, ls *MockListingService, us *MockUserService) {
				ls.GetByIDWithSellerForUpdateTxFunc = func(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) (*listing.ListingWithSeller, error) {
					return &listing.ListingWithSeller{
						ID:           uuid.New(),
						SellerID:     uuid.New(),
						Quantity:     10,
						UserIsFrozen: true, // Seller is frozen
					}, nil
				}
			},
			expectErr: true,
		},
		{
			name: "rollback on insufficient quantity",
			setupMocks: func(repo *MockRepository, ls *MockListingService, us *MockUserService) {
				ls.GetByIDWithSellerForUpdateTxFunc = func(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) (*listing.ListingWithSeller, error) {
					return &listing.ListingWithSeller{
						ID:           uuid.New(),
						SellerID:     uuid.New(),
						Quantity:     1, // Not enough quantity
						UserIsFrozen: false,
					}, nil
				}
			},
			expectErr: true,
		},
		{
			name: "rollback on listing update failure",
			setupMocks: func(repo *MockRepository, ls *MockListingService, us *MockUserService) {
				ls.GetByIDWithSellerForUpdateTxFunc = func(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) (*listing.ListingWithSeller, error) {
					return &listing.ListingWithSeller{
						ID:           uuid.New(),
						SellerID:     uuid.New(),
						Quantity:     10,
						UserIsFrozen: false,
						Price:        20.0,
					}, nil
				}
				ls.UpdateTxFunc = func(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, sID uuid.UUID, req *listing.UpdateListingRequest) (*listing.Listing, error) {
					return nil, errors.New("database error")
				}
			},
			expectErr: true,
		},
		{
			name: "rollback on order creation failure",
			setupMocks: func(repo *MockRepository, ls *MockListingService, us *MockUserService) {
				ls.GetByIDWithSellerForUpdateTxFunc = func(ctx context.Context, tx *sqlx.Tx, id uuid.UUID) (*listing.ListingWithSeller, error) {
					return &listing.ListingWithSeller{
						ID:           uuid.New(),
						SellerID:     uuid.New(),
						Quantity:     10,
						UserIsFrozen: false,
						Price:        20.0,
					}, nil
				}
				ls.UpdateTxFunc = func(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, sID uuid.UUID, req *listing.UpdateListingRequest) (*listing.Listing, error) {
					return &listing.Listing{}, nil
				}
				repo.CreateTxFunc = func(ctx context.Context, tx *sqlx.Tx, order *Order) error {
					return errors.New("constraint violation")
				}
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("Failed to create sqlmock: %v", err)
			}
			defer db.Close()

			sqlxDB := sqlx.NewDb(db, "sqlmock")

			// Expect transaction to begin and then rollback
			mock.ExpectBegin()
			mock.ExpectRollback()

			mockRepo := &MockRepository{}
			mockListingService := &MockListingService{}
			mockUserService := &MockUserService{}

			// Apply test-specific mocks
			tt.setupMocks(mockRepo, mockListingService, mockUserService)

			service := NewService(mockRepo, sqlxDB, slog.Default(), mockListingService, mockUserService)

			req := &CreateOrderRequest{
				ListingID: uuid.New(),
				Quantity:  2,
			}

			order, err := service.Create(context.Background(), uuid.New(), req)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if tt.expectErr && order != nil {
				t.Error("Expected nil order on error")
			}

			// Verify transaction was rolled back
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("Unfulfilled sqlmock expectations: %v", err)
			}
		})
	}
}

// TestExecTx_TransactionBehavior tests the transaction helper function
func TestExecTx_TransactionBehavior(t *testing.T) {
	t.Run("successful commit", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("Failed to create sqlmock: %v", err)
		}
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")

		mock.ExpectBegin()
		mock.ExpectCommit()

		executed := false
		err = dbutils.ExecTx(context.Background(), sqlxDB, func(tx *sqlx.Tx) error {
			executed = true
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !executed {
			t.Error("Transaction function was not executed")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("rollback on error", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("Failed to create sqlmock: %v", err)
		}
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")

		mock.ExpectBegin()
		mock.ExpectRollback()

		expectedErr := errors.New("something went wrong")
		err = dbutils.ExecTx(context.Background(), sqlxDB, func(tx *sqlx.Tx) error {
			return expectedErr
		})

		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})

	t.Run("begin failure", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("Failed to create sqlmock: %v", err)
		}
		defer db.Close()

		sqlxDB := sqlx.NewDb(db, "sqlmock")

		mock.ExpectBegin().WillReturnError(sql.ErrConnDone)

		err = dbutils.ExecTx(context.Background(), sqlxDB, func(tx *sqlx.Tx) error {
			t.Error("This should not be called")
			return nil
		})

		if err == nil {
			t.Error("Expected error when begin fails")
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("Unfulfilled expectations: %v", err)
		}
	})
}

