package usecase

import (
	"context"

	"github.com/darkphotonKN/seeyoulatte-app/services/order-service/internal/order/domain"
	"github.com/google/uuid"
)

type TransitionOrderUC struct {
	orderRepo domain.OrderRepository
	ledger    LedgerService  // narrow ISP interface, only methods that THIS usecase needs
	publisher EventPublisher // narrow ISP interface, only publishes state changed
}

type LedgerService interface{}
type EventPublisher interface{}

func NewTransitionOrderUC(repo domain.OrderRepository, ledger LedgerService, pub EventPublisher) *TransitionOrderUC {
	return &TransitionOrderUC{orderRepo: repo, ledger: ledger, publisher: pub}
}

func (uc *TransitionOrderUC) Handle(ctx context.Context, orderID uuid.UUID, target domain.OrderState, actor string) (*domain.Order, error) {

	return nil, nil
}
