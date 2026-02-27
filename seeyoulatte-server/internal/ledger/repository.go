package ledger

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// repository implements the append-only ledger storage
// CRITICAL: This repository MUST NOT have Update or Delete methods
// The ledger is immutable - corrections are made via reversal entries
type repository struct {
	db *sqlx.DB
}

// NewRepository creates a new ledger repository
func NewRepository(db *sqlx.DB) *repository {
	return &repository{db: db}
}

// Create inserts a new ledger entry
// This is the ONLY write operation allowed on the ledger
func (r *repository) Create(ctx context.Context, entry *LedgerEntry) error {
	query := `
		INSERT INTO ledger_entries (
			order_id, entry_type, amount, actor_id, actor_type, notes
		) VALUES (
			$1, $2, $3, $4, $5, $6
		) RETURNING id, created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		entry.OrderID,
		entry.EntryType,
		entry.Amount,
		entry.ActorID,
		entry.ActorType,
		entry.Notes,
	).Scan(&entry.ID, &entry.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create ledger entry: %w", err)
	}

	return nil
}

// GetByID retrieves a single ledger entry by ID
func (r *repository) GetByID(ctx context.Context, id int) (*LedgerEntry, error) {
	var entry LedgerEntry
	query := `
		SELECT
			id, order_id, entry_type, amount,
			actor_id, actor_type, notes, created_at
		FROM ledger_entries
		WHERE id = $1
	`

	err := r.db.GetContext(ctx, &entry, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("ledger entry not found with id %d", id)
		}
		return nil, fmt.Errorf("failed to get ledger entry: %w", err)
	}

	return &entry, nil
}

// GetByOrderID retrieves all ledger entries for a specific order
// Returns entries in chronological order (oldest first)
func (r *repository) GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]LedgerEntry, error) {
	var entries []LedgerEntry
	query := `
		SELECT
			id, order_id, entry_type, amount,
			actor_id, actor_type, notes, created_at
		FROM ledger_entries
		WHERE order_id = $1
		ORDER BY created_at ASC, id ASC
	`

	err := r.db.SelectContext(ctx, &entries, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get ledger entries for order %s: %w", orderID, err)
	}

	return entries, nil
}

// GetOrderBalance calculates the escrow balance for an order
// Follows the formula from SPECIFICATION.md:
// ESCROW entries add to balance, PAYOUT/REFUND/REVERSAL subtract
func (r *repository) GetOrderBalance(ctx context.Context, orderID uuid.UUID) (*BalanceCalculation, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN entry_type = 'ESCROW' THEN amount ELSE 0 END), 0) as total_escrow,
			COALESCE(SUM(CASE WHEN entry_type = 'PAYOUT' THEN amount ELSE 0 END), 0) as total_payout,
			COALESCE(SUM(CASE WHEN entry_type = 'REFUND' THEN amount ELSE 0 END), 0) as total_refund,
			COALESCE(SUM(CASE WHEN entry_type = 'REVERSAL' THEN amount ELSE 0 END), 0) as total_reversal,
			COALESCE(SUM(
				CASE
					WHEN entry_type = 'ESCROW' THEN amount
					WHEN entry_type IN ('PAYOUT', 'REFUND', 'REVERSAL') THEN -amount
					ELSE 0
				END
			), 0) as escrow_balance
		FROM ledger_entries
		WHERE order_id = $1
	`

	var calc BalanceCalculation
	calc.OrderID = orderID

	row := r.db.QueryRowContext(ctx, query, orderID)
	err := row.Scan(
		&calc.TotalEscrow,
		&calc.TotalPayout,
		&calc.TotalRefund,
		&calc.TotalReversal,
		&calc.EscrowBalance,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate order balance: %w", err)
	}

	return &calc, nil
}

// GetEntriesByType retrieves all entries of a specific type for an order
func (r *repository) GetEntriesByType(ctx context.Context, orderID uuid.UUID, entryType EntryType) ([]LedgerEntry, error) {
	var entries []LedgerEntry
	query := `
		SELECT
			id, order_id, entry_type, amount,
			actor_id, actor_type, notes, created_at
		FROM ledger_entries
		WHERE order_id = $1 AND entry_type = $2
		ORDER BY created_at ASC, id ASC
	`

	err := r.db.SelectContext(ctx, &entries, query, orderID, entryType)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s entries for order %s: %w", entryType, orderID, err)
	}

	return entries, nil
}

// CountEntriesByType returns the count of entries of a specific type for an order
func (r *repository) CountEntriesByType(ctx context.Context, orderID uuid.UUID, entryType EntryType) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM ledger_entries
		WHERE order_id = $1 AND entry_type = $2
	`

	err := r.db.GetContext(ctx, &count, query, orderID, entryType)
	if err != nil {
		return 0, fmt.Errorf("failed to count %s entries: %w", entryType, err)
	}

	return count, nil
}