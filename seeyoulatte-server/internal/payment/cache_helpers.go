package payment

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/darkphotonKN/seeyoulatte-app/internal/interfaces"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Cache helper methods for payment service

// AddCacheUserIdToCustomerId adds/sets the mapping between customerId and userId in cache
func (s *service) AddCacheUserIdToCustomerId(ctx context.Context, userId uuid.UUID, customerId string) error {
	if s.cacheClient == nil {
		return nil // Cache is optional, skip if not configured
	}

	s.logger.Info("updating customerId to userId mapping in cache",
		slog.String("customer_id", customerId),
		slog.String("user_id", userId.String()))

	key := s.cacheClient.GetUserIdFromCustomerIdKey(customerId)
	err := s.cacheClient.Set(ctx, key, userId.String(), 0)

	if err != nil {
		return fmt.Errorf("failed to cache userId to customerId mapping: %w", err)
	}

	return nil
}

// AddCacheCustomerIdToUserId adds/sets the mapping between userId and customerId in cache
func (s *service) AddCacheCustomerIdToUserId(ctx context.Context, customerId string, userId uuid.UUID) error {
	if s.cacheClient == nil {
		return nil // Cache is optional, skip if not configured
	}

	key := s.cacheClient.GetCustomerIdFromUserIdKey(userId.String())
	err := s.cacheClient.Set(ctx, key, customerId, 0)

	if err != nil {
		return fmt.Errorf("failed to cache customerId to userId mapping: %w", err)
	}

	return nil
}

// GetCachedCustomerIdFromUserId gets cached customerId with the userId
func (s *service) GetCachedCustomerIdFromUserId(ctx context.Context, userId uuid.UUID) (string, error) {
	if s.cacheClient == nil {
		// No cache, fall back to database
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

	// If not in cache, get from database
	if err == redis.Nil {
		s.logger.Debug("customerId not found in cache, fetching from database",
			slog.String("user_id", userId.String()))

		user, err := s.userService.GetByID(ctx, userId)
		if err != nil {
			return "", err
		}

		if user.StripeCustomerID == nil {
			return "", fmt.Errorf("user has no payment processor customer ID")
		}

		// Store in cache for next time
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

// GetCachedUserIdByCustomerId gets cached userId by customerId
func (s *service) GetCachedUserIdByCustomerId(ctx context.Context, customerID string) (uuid.UUID, error) {
	if s.cacheClient == nil {
		// No cache, fall back to database
		user, err := s.userService.GetByStripeCustomerID(ctx, customerID)
		if err != nil {
			return uuid.Nil, err
		}
		return user.ID, nil
	}

	key := s.cacheClient.GetUserIdFromCustomerIdKey(customerID)
	userIdStr, err := s.cacheClient.Get(ctx, key)

	// If not in cache, get from database
	if err == redis.Nil {
		user, err := s.userService.GetByStripeCustomerID(ctx, customerID)
		if err != nil {
			s.logger.Error("error getting user by customerId",
				slog.String("customer_id", customerID),
				slog.String("error", err.Error()))
			return uuid.Nil, err
		}

		// Store in cache for next time
		s.cacheClient.Set(ctx, key, user.ID.String(), 0)

		return user.ID, nil
	}

	if err != nil {
		s.logger.Error("error getting userId from cache",
			slog.String("customer_id", customerID),
			slog.String("error", err.Error()))
		return uuid.Nil, err
	}

	// Convert string to UUID
	userId, err := uuid.Parse(userIdStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid UUID in cache: %w", err)
	}

	return userId, nil
}


// GetPaymentProcessorData retrieves payment processor data from cache
func (s *service) GetPaymentProcessorData(ctx context.Context, customerId string) (*PaymentProcessorCacheData, error) {
	if s.cacheClient == nil {
		return nil, fmt.Errorf("cache not available")
	}

	key := s.cacheClient.GetCustomerDataKey(customerId)

	// Try to get from cache
	var cacheData PaymentProcessorCacheData
	err := s.cacheClient.GetJSON(ctx, key, &cacheData)

	if err != nil {
		return nil, err
	}

	return &cacheData, nil
}

// SetCacheClient sets the cache client for the service
func (s *service) SetCacheClient(client interfaces.Cache) {
	s.cacheClient = client
}