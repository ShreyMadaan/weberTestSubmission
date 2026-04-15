package main

import (
"context"
"fmt"
"log"
"time"

"github.com/jackc/pgx/v5/pgxpool"
"github.com/your-username/auctioncore/internal/config"
)

func main() {
cfg := config.Load()

dsn := fmt.Sprintf(
"postgres://%s:%s@%s:%s/%s?sslmode=%s",
cfg.PostgresUser,
cfg.PostgresPass,
cfg.PostgresHost,
cfg.PostgresPort,
cfg.PostgresDB,
cfg.PostgresSSL,
)

ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

pool, err := pgxpool.New(ctx, dsn)
if err != nil {
log.Fatalf("connect db: %v", err)
}
defer pool.Close()

log.Println("seeding database...")

if err := seedAuctionHouses(ctx, pool); err != nil {
log.Fatalf("seed auction houses: %v", err)
}
if err := seedAuctionsAndLots(ctx, pool); err != nil {
log.Fatalf("seed auctions/lots: %v", err)
}
if err := seedBiddersAndBids(ctx, pool); err != nil {
log.Fatalf("seed bidders/bids: %v", err)
}

log.Println("seed completed")
}

func seedAuctionHouses(ctx context.Context, pool *pgxpool.Pool) error {
// TODO: insert 5 auction houses with realistic slugs and buyer premium
return nil
}

func seedAuctionsAndLots(ctx context.Context, pool *pgxpool.Pool) error {
// TODO: insert 20 auctions and 200 lots linked to those auctions with schedules and closing times
return nil
}

func seedBiddersAndBids(ctx context.Context, pool *pgxpool.Pool) error {
// TODO: insert 500 bidders and ~2000 historical bids with realistic increment patterns
return nil
}
