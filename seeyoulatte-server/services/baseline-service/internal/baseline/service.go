package baseline

import (
	"context"
	"fmt"

	commonbroker "github.com/darkphotonKN/seeyoulatte-app/common/broker"
)

type service struct {
	publishCh commonbroker.Publisher
}

func NewService(publishCh commonbroker.Publisher) *service {
	return &service{publishCh: publishCh}
}

func (s *service) Ping(_ context.Context, msg string) string {
	return fmt.Sprintf("pong: %s", msg)
}
