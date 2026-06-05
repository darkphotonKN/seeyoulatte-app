package usecase

import (
	"context"
)

type CreateOrderUC struct {
	orderRepo OrderRepository
}

type OrderRepository interface {
	// TODO: need to complete
	CreateOrder(ctx context.Context) error
}

func NewCreateOrderUC(orderRepo OrderRepository) *CreateOrderUC {
	return &CreateOrderUC{
		orderRepo: orderRepo,
	}
}

func (uc *CreateOrderUC) Handle(ctx context.Context, req) {
}
