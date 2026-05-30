package ledger

import (
	"context"

	"github.com/google/uuid"
)

type Facade struct {
	escrow *CreateEscrowEntryUC
	payout *CreatePayoutEntryUC
	refund *CreateRefundEntryUC
}

func NewFacade(escrow *CreateEscrowEntryUC, payout *CreatePayoutEntryUC, refund *CreateRefundEntryUC) *Facade {
	return &Facade{
		escrow: escrow,
		payout: payout,
		refund: refund,
	}
}

// TODO: temps, remove later when implemented
type CreateEscrowEntryUC struct{}

func (e *CreateEscrowEntryUC) Handle(ctx context.Context, orderID uuid.UUID, amount float64, actorID uuid.UUID) error {
	return nil
}

type CreatePayoutEntryUC struct{}

func (p *CreatePayoutEntryUC) Handle(ctx context.Context, orderID uuid.UUID, amount float64) error {
	return nil
}

type CreateRefundEntryUC struct{}

func (r *CreateRefundEntryUC) Handle(ctx context.Context, orderID uuid.UUID, amount float64, notes string) error {
	return nil
}

func (f *Facade) CreateEscrowEntry(ctx context.Context, orderID uuid.UUID, amount float64, actorID uuid.UUID) error {
	return f.escrow.Handle(ctx, orderID, amount, actorID)
}
func (f *Facade) CreatePayoutEntry(ctx context.Context, orderID uuid.UUID, amount float64) error {
	return f.payout.Handle(ctx, orderID, amount)
}
func (f *Facade) CreateRefundEntry(ctx context.Context, orderID uuid.UUID, amount float64, notes string) error {
	return f.refund.Handle(ctx, orderID, amount, notes)
}
