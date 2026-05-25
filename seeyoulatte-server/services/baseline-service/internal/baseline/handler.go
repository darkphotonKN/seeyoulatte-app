package baseline

import (
	"context"
	"os"

	pb "github.com/darkphotonKN/seeyoulatte-app/common/api/proto/baseline"
)

// Service is the narrow interface the handler depends on.
type Service interface {
	Ping(ctx context.Context, msg string) string
}

type Handler struct {
	pb.UnimplementedBaselineServiceServer
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// Ping is implemented end-to-end so the service can be smoke-tested via
// grpcurl once the stack is up:
//
//	grpcurl -plaintext -d '{"message":"hi"}' localhost:7302 baseline.BaselineService/Ping
func (h *Handler) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	hostname, _ := os.Hostname()
	reply := h.service.Ping(ctx, req.Message)
	return &pb.PingResponse{
		Reply:    reply,
		ServedBy: hostname,
	}, nil
}
