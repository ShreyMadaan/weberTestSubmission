package app

import (
"github.com/jackc/pgx/v5/pgxpool"
"github.com/redis/go-redis/v9"

"github.com/your-username/auctioncore/internal/config"
"github.com/your-username/auctioncore/internal/httpserver"
"github.com/your-username/auctioncore/internal/storage"
)

type App struct {
Config *config.Config
DB     *pgxpool.Pool
Redis  *redis.Client
Router httpserver.Router
}

func New() (*App, error) {
cfg := config.Load()

db, err := storage.NewPostgres(cfg)
if err != nil {
return nil, err
}

rdb, err := storage.NewRedis(cfg)
if err != nil {
db.Close()
return nil, err
}

router := httpserver.NewRouter()

return &App{
Config: cfg,
DB:     db,
Redis:  rdb,
Router: router,
}, nil
}

func (a *App) Close() {
if a.Redis != nil {
_ = a.Redis.Close()
}
if a.DB != nil {
a.DB.Close()
}
}
