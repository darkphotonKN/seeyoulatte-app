# Order Service Testing Guide

## Overview

This document explains the best practices for unit testing the order service's `Create` method, which involves database transactions and multiple service dependencies.

## Key Testing Concepts

### 1. Mock Interfaces

We define mock implementations for all dependencies:
- `MockRepository` - Mocks the order repository
- `MockListingService` - Mocks listing service operations
- `MockUserService` - Mocks user validation

Each mock uses function fields that can be customized per test:

```go
type MockRepository struct {
    CreateTxFunc func(ctx context.Context, tx *sqlx.Tx, order *Order) error
    // ... other methods
}
```

### 2. sqlmock for Transaction Testing

We use `github.com/DATA-DOG/go-sqlmock` to mock database transactions:

```go
// Create mock database
db, mock, err := sqlmock.New()
sqlxDB := sqlx.NewDb(db, "sqlmock")

// Set expectations
mock.ExpectBegin()
mock.ExpectCommit() // or mock.ExpectRollback() for error cases
```

### 3. Testing Transaction Flow

The `Create` method uses `dbutils.ExecTx` which handles:
- Beginning the transaction
- Executing the provided function
- Committing on success
- Rolling back on error

Our tests verify all these behaviors.

## Test Structure

### Successful Order Creation Test

Tests the happy path:
1. User is not frozen
2. Transaction begins
3. Listing is fetched with lock (FOR UPDATE)
4. Quantity is sufficient
5. Seller is not frozen
6. Buyer is not the seller
7. Listing quantity is decremented
8. Order is created with correct amount calculation
9. Transaction commits

### Error Scenarios

Each error case verifies transaction rollback:

1. **Buyer Frozen** - No transaction started
2. **Seller Frozen** - Transaction rolls back
3. **Insufficient Quantity** - Transaction rolls back
4. **Self-Purchase** - Transaction rolls back
5. **Listing Update Failure** - Transaction rolls back
6. **Order Creation Failure** - Transaction rolls back

### Transaction Helper Tests

We also test the `ExecTx` function directly:
- Successful commit
- Rollback on error
- Begin failure
- Commit failure

## Running the Tests

```bash
# Run all order tests
go test ./internal/order -v

# Run specific test
go test ./internal/order -run TestCreateOrder_SuccessfulFlow -v

# Run with coverage
go test ./internal/order -cover -v
```

## Best Practices Applied

1. **Interface Segregation** - Each mock only implements what's needed
2. **Dependency Injection** - All dependencies injected via constructor
3. **Transaction Verification** - sqlmock ensures proper transaction handling
4. **Data Validation** - Mock functions verify correct data is passed
5. **Error Handling** - All error paths tested including rollbacks
6. **Isolation** - Each test is independent with its own mocks

## Example: Testing a Transactional Method

```go
func TestServiceMethod_WithTransaction(t *testing.T) {
    // 1. Create sqlmock
    db, mock, err := sqlmock.New()
    defer db.Close()
    sqlxDB := sqlx.NewDb(db, "sqlmock")

    // 2. Set transaction expectations
    mock.ExpectBegin()
    mock.ExpectCommit() // or ExpectRollback for error cases

    // 3. Create mocks with custom behavior
    mockRepo := &MockRepository{
        CreateTxFunc: func(ctx context.Context, tx *sqlx.Tx, order *Order) error {
            // Verify data and return result
            return nil
        },
    }

    // 4. Create service with mocks
    service := NewService(mockRepo, sqlxDB, logger, ...)

    // 5. Execute method
    result, err := service.MethodUnderTest(...)

    // 6. Verify expectations met
    if err := mock.ExpectationsWereMet(); err != nil {
        t.Errorf("Unfulfilled expectations: %v", err)
    }
}
```

## Key Insights

### Why Mock the Transaction?

1. **Speed** - No actual database needed
2. **Reliability** - Tests don't depend on database state
3. **Coverage** - Can test error scenarios easily
4. **Verification** - Ensures correct transaction boundaries

### What to Verify?

1. **Transaction Flow** - Begin, Commit/Rollback called correctly
2. **Data Integrity** - Correct data passed between services
3. **Error Propagation** - Errors cause rollback
4. **Business Rules** - All validations enforced
5. **State Changes** - Order state, quantities, amounts correct

### Common Pitfalls to Avoid

1. Not verifying transaction rollback on errors
2. Forgetting to test concurrent access scenarios
3. Not validating the data passed to mocks
4. Missing edge cases like self-purchase
5. Not testing the transaction helper function separately

## Integration vs Unit Tests

This approach focuses on unit tests with mocked dependencies. For integration tests:
- Use a real test database
- Test actual SQL queries
- Verify constraints and locks
- Test concurrent operations

Both types of tests are valuable and complementary.