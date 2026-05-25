package payment

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/darkphotonKN/seeyoulatte-app/services/api-gateway/internal/interfaces"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Cache helper methods for payment service

// AddCacheUserIdToCustomerId adds/sets the mapping between customerId and userId in cache
func (s *service) AddCacheUserIdToCustomerId(ctx context.Context, userId uuid.UUID, customerId string) error {
	if s.cacheClient == nil {
		return nil
	}

	s.logger.Info("updating customerId to userId mapping in cache",
		slog.String("customer_id", customerId),
		slog.String("user_id", userId.String()))

	key := s.cacheClient.GetUserIdFromCustomerIdKey(customerId)
	if err := s.cacheClient.Set(ctx, key, userId.String(), 0); err != nil {
		return fmt.Errorf("failed to cache userId to customerId mapping: %w", err)
	}
	return nil
}

// AddCacheCustomerIdToUserId adds/sets the mapping between userId and customerId in cache
func (s *service) AddCacheCustomerIdToUserId(ctx context.Context, customerId string, userId uuid.UUID) error {
	if s.cacheClient == nil {
		return nil
	}
	key := s.cacheClient.GetCustomerIdFromUserIdKey(userId.String())
	if err := s.cacheClient.Set(ctx, key, customerId, 0); err != nil {
		return fmt.Errorf("failed to cache customerId to userId mapping: %w", err)
	}
	return nil
}

// GetCachedCustomerIdFromUserId gets cached customerId for a userId. Falls back
// to the auth-service gRPC lookup if cache is missing or unconfigured.
func (s *service) GetCachedCustomerIdFromUserId(ctx context.Context, userId uuid.UUID) (string, error) {
	if s.cacheClient == nil {
		user, err := s.userService.GetByID(ctx, userId)
		if err != nil {
			return "", err
		}
		if user.StripeCustomerID == nil {
			return "", fmt.Errorf("user has no payment processor customer ID")
		}
		return *user.StripeCustomerID, nil
	}

	key := s.cacheClient.GetCustomerIdFromUserIdKey(userId.String())
	customerId, err := s.cacheClient.Get(ctx, key)
	if err == redis.Nil {
		s.logger.Debug("customerId not found in cache, fetching from auth-service",
			slog.String("user_id", userId.String()))
		user, err := s.userService.GetByID(ctx, userId)
		if err != nil {
			return "", err
		}
		if user.StripeCustomerID == nil {
			return "", fmt.Errorf("user has no payment processor customer ID")
		}
		s.cacheClient.Set(ctx, key, *user.StripeCustomerID, 0)
		return *user.StripeCustomerID, nil
	}
	if err != nil {
		s.logger.Error("error getting customerId from cache",
			slog.String("user_id", userId.String()),
			slog.String("error", err.Error()))
		return "", err
	}
	return customerId, nil
}

// GetCachedUserIdByCustomerId gets cached userId by customerId.
func (s *service) GetCachedUserIdByCustomerId(ctx context.Context, customerID string) (uuid.UUID, error) {
	if s.cacheClient == nil {
		user, err := s.userService.GetByStripeCustomerID(ctx, customerID)
		if err != nil {
			return uuid.Nil, err
		}
		return user.ID, nil
	}

	key := s.cacheClient.GetUserIdFromCustomerIdKey(customerID)
	userIdStr, err := s.cacheClient.Get(ctx, key)
	if err == redis.Nil {
		user, err := s.userService.GetByStripeCustomerID(ctx, customerID)
		if err != nil {
			s.logger.Error("error getting user by customerId",
				slog.String("customer_id", customerID),
				slog.String("error", err.Error()))
			return uuid.Nil, err
		}
		s.cacheClient.Set(ctx, key, user.ID.String(), 0)
		return user.ID, nil
	}
	if err != nil {
		s.logger.Error("error getting userId from cache",
			slog.String("customer_id", customerID),
			slog.String("error", err.Error()))
		return uuid.Nil, err
	}
	return uuid.Parse(userIdStr)
}

// GetPaymentProcessorData retrieves payment processor data from cache
func (s *service) GetPaymentProcessorData(ctx context.Context, customerId string) (*PaymentProcessorCacheData, error) {
	if s.cacheClient == nil {
		return nil, fmt.Errorf("cache not available")
	}
	key := s.cacheClient.GetCustomerDataKey(customerId)
	var cacheData PaymentProcessorCacheData
	if err := s.cacheClient.GetJSON(ctx, key, &cacheData); err != nil {
		return nil, err
	}
	return &cacheData, nil
}

// SetCacheClient sets the cache client for the service
func (s *service) SetCacheClient(client interfaces.Cache) {
	s.cacheClient = client
}
