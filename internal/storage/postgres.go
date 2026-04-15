package storage

import (
"context"
"fmt"
"time"

"github.com/jackc/pgx/v5/pgxpool"
"github.com/your-username/auctioncore/internal/config"
)

func NewPostgres(cfg *config.Config) (*pgxpool.Pool, error) {
dsn := fmt.Sprintf(
"postgres://%s:%s@%s:%s/%s?sslmode=%s",
cfg.PostgresUser,
cfg.PostgresPass,
cfg.PostgresHost,
cfg.PostgresPort,
cfg.PostgresDB,
cfg.PostgresSSL,
)

poolCfg, err := pgxpool.ParseConfig(dsn)
if err != nil {
return nil, err
}

poolCfg.MaxConns = 20
poolCfg.MinConns = 2
poolCfg.MaxConnLifetime = 30 * time.Minute
poolCfg.MaxConnIdleTime = 5 * time.Minute

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
if err != nil {
return nil, err
}

if err := pool.Ping(ctx); err != nil {
pool.Close()
return nil, err
}

return pool, nil
}
