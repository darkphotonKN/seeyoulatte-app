package ledger

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// Repository interface defines what the service needs from the repository
// Following ISP - the service defines what it needs
type Repository interface {
	Create(ctx context.Context, entry *LedgerEntry) error
	GetByID(ctx context.Context, id int) (*LedgerEntry, error)
	GetByOrderID(ctx context.Context, orderID uuid.UUID) ([]LedgerEntry, error)
	GetOrderBalance(ctx context.Context, orderID uuid.UUID) (*BalanceCalculation, error)
	GetEntriesByType(ctx context.Context, orderID uuid.UUID, entryType EntryType) ([]LedgerEntry, error)
	CountEntriesByType(ctx context.Context, orderID uuid.UUID, entryType EntryType) (int, error)
}

// service implements the ledger business logic
type service struct {
	repo   Repository
	logger *slog.Logger
}

// NewService creates a new ledger service
func NewService(repo Repository, logger *slog.Logger) *service {
	return &service{
		repo:   repo,
		logger: logger,
	}
}

// CreateEscrowEntry creates an ESCROW entry when payment is confirmed
// This represents money entering the platform's hold
func (s *service) CreateEscrowEntry(ctx context.Context, orderID uuid.UUID, amount float64, actorID uuid.UUID) error {
	if amount <= 0 {
		return fmt.Errorf("escrow amount must be positive, got %f", amount)
	}

	actorType := ActorTypeBuyer
	entry := &LedgerEntry{
		OrderID:   orderID,
		EntryType: EntryTypeEscrow,
		Amount:    amount,
		ActorID:   &actorID,
		ActorType: &actorType,
	}

	err := s.repo.Create(ctx, entry)
	if err != nil {
		s.logger.Error("failed to create escrow entry",
			"orderID", orderID,
			"amount", amount,
			"error", err)
		return fmt.Errorf("failed to create escrow entry: %w", err)
	}

	s.logger.Info("escrow entry created",
		"orderID", orderID,
		"amount", amount,
		"entryID", entry.ID)

	return nil
}

// CreatePayoutEntry creates a PAYOUT entry when money is released to the seller
func (s *service) CreatePayoutEntry(ctx context.Context, orderID uuid.UUID, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("payout amount must be positive, got %f", amount)
	}

	// Check if there's sufficient escrow balance
	balance, err := s.repo.GetOrderBalance(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to check escrow balance: %w", err)
	}

	if balance.EscrowBalance < amount {
		return fmt.Errorf("insufficient escrow balance: have %f, need %f", balance.EscrowBalance, amount)
	}

	actorType := ActorTypeSystem
	notes := "Order completed - payout to seller"
	entry := &LedgerEntry{
		OrderID:   orderID,
		EntryType: EntryTypePayout,
		Amount:    amount,
		ActorType: &actorType,
		Notes:     &notes,
	}

	err = s.repo.Create(ctx, entry)
	if err != nil {
		s.logger.Error("failed to create payout entry",
			"orderID", orderID,
			"amount", amount,
			"error", err)
		return fmt.Errorf("failed to create payout entry: %w", err)
	}

	s.logger.Info("payout entry created",
		"orderID", orderID,
		"amount", amount,
		"entryID", entry.ID)

	return nil
}

// CreateRefundEntry creates a REFUND entry when money is returned to the buyer
func (s *service) CreateRefundEntry(ctx context.Context, orderID uuid.UUID, amount float64, notes string) error {
	if amount <= 0 {
		return fmt.Errorf("refund amount must be positive, got %f", amount)
	}

	// Check if there's sufficient escrow balance
	balance, err := s.repo.GetOrderBalance(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to check escrow balance: %w", err)
	}

	if balance.EscrowBalance < amount {
		return fmt.Errorf("insufficient escrow balance for refund: have %f, need %f", balance.EscrowBalance, amount)
	}

	actorType := ActorTypeSystem
	if notes == "" {
		notes = "Order cancelled - refund to buyer"
	}

	entry := &LedgerEntry{
		OrderID:   orderID,
		EntryType: EntryTypeRefund,
		Amount:    amount,
		ActorType: &actorType,
		Notes:     &notes,
	}

	err = s.repo.Create(ctx, entry)
	if err != nil {
		s.logger.Error("failed to create refund entry",
			"orderID", orderID,
			"amount", amount,
			"error", err)
		return fmt.Errorf("failed to create refund entry: %w", err)
	}

	s.logger.Info("refund entry created",
		"orderID", orderID,
		"amount", amount,
		"notes", notes,
		"entryID", entry.ID)

	return nil
}

// CreateReversalEntry creates a REVERSAL entry to correct a previous erroneous entry
// Per SPECIFICATION.md: corrections are made via new entries, not updates
func (s *service) CreateReversalEntry(ctx context.Context, orderID uuid.UUID, amount float64, notes string, actorID uuid.UUID) error {
	if amount <= 0 {
		return fmt.Errorf("reversal amount must be positive, got %f", amount)
	}

	if notes == "" {
		return fmt.Errorf("reversal entries must include notes explaining the correction")
	}

	actorType := ActorTypeAdmin
	entry := &LedgerEntry{
		OrderID:   orderID,
		EntryType: EntryTypeReversal,
		Amount:    amount,
		ActorID:   &actorID,
		ActorType: &actorType,
		Notes:     &notes,
	}

	err := s.repo.Create(ctx, entry)
	if err != nil {
		s.logger.Error("failed to create reversal entry",
			"orderID", orderID,
			"amount", amount,
			"error", err)
		return fmt.Errorf("failed to create reversal entry: %w", err)
	}

	s.logger.Info("reversal entry created",
		"orderID", orderID,
		"amount", amount,
		"notes", notes,
		"entryID", entry.ID)

	return nil
}

// CalculateOrderBalance calculates the current escrow balance for an order
// Balance > 0: funds still held, Balance = 0: fully disbursed
func (s *service) CalculateOrderBalance(ctx context.Context, orderID uuid.UUID) (*BalanceCalculation, error) {
	balance, err := s.repo.GetOrderBalance(ctx, orderID)
	if err != nil {
		s.logger.Error("failed to calculate order balance",
			"orderID", orderID,
			"error", err)
		return nil, fmt.Errorf("failed to calculate order balance: %w", err)
	}

	return balance, nil
}

// GetOrderLedger retrieves all ledger entries for an order
func (s *service) GetOrderLedger(ctx context.Context, orderID uuid.UUID) ([]LedgerEntry, error) {
	entries, err := s.repo.GetByOrderID(ctx, orderID)
	if err != nil {
		s.logger.Error("failed to get order ledger",
			"orderID", orderID,
			"error", err)
		return nil, fmt.Errorf("failed to get order ledger: %w", err)
	}

	return entries, nil
}

// ValidateEntryType checks if an entry type is valid
func (s *service) ValidateEntryType(entryType string) error {
	et := EntryType(entryType)
	if !et.IsValid() {
		return fmt.Errorf("invalid entry type: %s", entryType)
	}
	return nil
}

// HasEscrowEntry checks if an order has at least one escrow entry
func (s *service) HasEscrowEntry(ctx context.Context, orderID uuid.UUID) (bool, error) {
	count, err := s.repo.CountEntriesByType(ctx, orderID, EntryTypeEscrow)
	if err != nil {
		return false, fmt.Errorf("failed to check for escrow entries: %w", err)
	}
	return count > 0, nil
}

// HasPayoutEntry checks if an order has been paid out
func (s *service) HasPayoutEntry(ctx context.Context, orderID uuid.UUID) (bool, error) {
	count, err := s.repo.CountEntriesByType(ctx, orderID, EntryTypePayout)
	if err != nil {
		return false, fmt.Errorf("failed to check for payout entries: %w", err)
	}
	return count > 0, nil
}

// HasRefundEntry checks if an order has been refunded
func (s *service) HasRefundEntry(ctx context.Context, orderID uuid.UUID) (bool, error) {
	count, err := s.repo.CountEntriesByType(ctx, orderID, EntryTypeRefund)
	if err != nil {
		return false, fmt.Errorf("failed to check for refund entries: %w", err)
	}
	return count > 0, nil
}