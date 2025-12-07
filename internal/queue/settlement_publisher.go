package queue

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// SettlementPublisher publishes transaction events to Redis for settlement processing
type SettlementPublisher struct {
	client *redis.Client
}

// NewSettlementPublisher creates a new settlement event publisher
func NewSettlementPublisher() *SettlementPublisher {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis:6379"
	}

	// Strip redis:// scheme if present
	if len(redisURL) > 8 && redisURL[:8] == "redis://" {
		redisURL = redisURL[8:]
	}

	client := redis.NewClient(&redis.Options{
		Addr:     redisURL,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Failed to connect to Redis at %s: %v", redisURL, err)
	} else {
		log.Printf("Successfully connected to Redis at %s", redisURL)
	}

	return &SettlementPublisher{
		client: client,
	}
}

// PublishTransaction publishes a transaction event for settlement processing
func (p *SettlementPublisher) PublishTransaction(ctx context.Context, merchantID int, amount int64, currency string, txID int) error {
	if p.client == nil {
		return fmt.Errorf("redis client not initialized")
	}

	merchantKey := strconv.Itoa(merchantID)
	txKeyValue := strconv.Itoa(txID)

	// Add merchant to pending set (merchants that need settlement check)
	if err := p.client.SAdd(ctx, "settlements:merchants:pending", merchantKey).Err(); err != nil {
		log.Printf("Failed to add merchant %s to pending set: %v", merchantKey, err)
		return err
	}

	// Increment merchant's unsettled amount
	key := "settlements:amounts:" + merchantKey
	if err := p.client.IncrBy(ctx, key, amount).Err(); err != nil {
		log.Printf("Failed to increment amount for merchant %s: %v", merchantKey, err)
		return err
	}

	// Add to merchant's transaction set (for tracking which transactions are included)
	txKey := "settlements:txns:" + merchantKey
	if err := p.client.SAdd(ctx, txKey, txKeyValue).Err(); err != nil {
		log.Printf("Failed to add transaction to merchant set: %v", err)
		return err
	}

	// Set expiry on amount key (30 days) to prevent stale data
	p.client.Expire(ctx, key, 30*24*time.Hour)
	p.client.Expire(ctx, txKey, 30*24*time.Hour)

	log.Printf("Published settlement event: merchant=%s, amount=%d, currency=%s, tx=%s",
		merchantKey, amount, currency, txKeyValue)

	return nil
}

// GetPendingMerchants returns list of merchants with pending settlements
func (p *SettlementPublisher) GetPendingMerchants(ctx context.Context) ([]string, error) {
	members, err := p.client.SMembers(ctx, "settlements:merchants:pending").Result()
	if err != nil {
		return nil, err
	}
	return members, nil
}

// GetMerchantAmount returns the unsettled amount for a merchant
func (p *SettlementPublisher) GetMerchantAmount(ctx context.Context, merchantID string) (int64, error) {
	key := "settlements:amounts:" + merchantID
	amountStr, err := p.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return 0, nil // No pending amount
	}
	if err != nil {
		return 0, err
	}

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return amount, nil
}

// ClearMerchantPending clears pending settlement data for a merchant after settlement
func (p *SettlementPublisher) ClearMerchantPending(ctx context.Context, merchantID string) error {
	pipe := p.client.Pipeline()

	// Remove from pending set
	pipe.SRem(ctx, "settlements:merchants:pending", merchantID)

	// Clear amount
	pipe.Del(ctx, "settlements:amounts:"+merchantID)

	// Clear transaction set
	pipe.Del(ctx, "settlements:txns:"+merchantID)

	_, err := pipe.Exec(ctx)
	return err
}

// Close closes the Redis connection
func (p *SettlementPublisher) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}
