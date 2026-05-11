package middleware

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func RateLimitInterceptor(rdb *redis.Client) grpc.UnaryServerInterceptor {
	rpmStr := os.Getenv("RATE_LIMIT_RPM")
	rpm, err := strconv.Atoi(rpmStr)
	if err != nil {
		log.Printf("Warning: RATE_LIMIT_RPM not set or invalid (%s), defaulting to 100", rpmStr)
		rpm = 100
	} else {
		log.Printf("Rate limiter initialized with %d RPM", rpm)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if rdb == nil {
			log.Printf("Rate limiter skipped: Redis client is nil")
			return handler(ctx, req)
		}

		p, ok := peer.FromContext(ctx)
		if !ok {
			log.Printf("Rate limiter skipped: could not get peer info from context")
			return handler(ctx, req)
		}

		ip, _, err := net.SplitHostPort(p.Addr.String())
		if err != nil {
			ip = p.Addr.String()
		}

		// Use a fixed window counter using Redis
		// Key: ratelimit:<ip>:<minute_timestamp>
		now := time.Now()
		minute := now.Unix() / 60
		key := fmt.Sprintf("ratelimit:%s:%d", ip, minute)

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			log.Printf("Rate limiter error for key %s: %v", key, err)
			return handler(ctx, req) // Best effort
		}

		if count == 1 {
			rdb.Expire(ctx, key, 2*time.Minute)
		}

		log.Printf("Rate limit check: IP=%s, Count=%d/%d, Key=%s", ip, count, rpm, key)

		if count > int64(rpm) {
			log.Printf("Rate limit exceeded for IP %s: %d > %d", ip, count, rpm)
			return nil, status.Errorf(codes.ResourceExhausted, "Rate limit exceeded. Max %d requests per minute. Try again later.", rpm)
		}

		return handler(ctx, req)
	}
}
