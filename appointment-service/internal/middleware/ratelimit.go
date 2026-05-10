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
		rpm = 100
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if rdb == nil {
			return handler(ctx, req)
		}

		p, ok := peer.FromContext(ctx)
		if !ok {
			return handler(ctx, req)
		}

		ip, _, err := net.SplitHostPort(p.Addr.String())
		if err != nil {
			ip = p.Addr.String()
		}

		// Use a sliding window counter using Redis
		now := time.Now()
		minute := now.Unix() / 60
		key := fmt.Sprintf("ratelimit:%s:%d", ip, minute)

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			log.Printf("Rate limiter error: %v", err)
			return handler(ctx, req) // Best effort
		}

		if count == 1 {
			rdb.Expire(ctx, key, 2*time.Minute)
		}

		if count > int64(rpm) {
			return nil, status.Errorf(codes.ResourceExhausted, "Rate limit exceeded. Max %d requests per minute. Try again later.", rpm)
		}

		return handler(ctx, req)
	}
}
