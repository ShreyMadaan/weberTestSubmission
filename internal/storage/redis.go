package storage

import (
"context"
"strconv"
"time"

"github.com/redis/go-redis/v9"
"github.com/your-username/auctioncore/internal/config"
)

func NewRedis(cfg *config.Config) (*redis.Client, error) {
db, err := strconv.Atoi(cfg.RedisDB)
if err != nil {
return nil, err
}

client := redis.NewClient(&redis.Options{
Addr:         cfg.RedisAddr,
Password:     cfg.RedisPassword,
DB:           db,
ReadTimeout:  3 * time.Second,
WriteTimeout: 3 * time.Second,
})

ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
defer cancel()

if err := client.Ping(ctx).Err(); err != nil {
return nil, err
}

return client, nil
}
