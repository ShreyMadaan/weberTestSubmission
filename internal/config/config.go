package config

import "os"

type Config struct {
AppEnv        string
HTTPPort      string
PostgresHost  string
PostgresPort  string
PostgresUser  string
PostgresPass  string
PostgresDB    string
PostgresSSL   string
RedisAddr     string
RedisPassword string
RedisDB       string
}

func getEnv(key, fallback string) string {
value := os.Getenv(key)
if value == "" {
return fallback
}
return value
}

func Load() *Config {
return &Config{
AppEnv:        getEnv("APP_ENV", "local"),
HTTPPort:      getEnv("HTTP_PORT", "8080"),
PostgresHost:  getEnv("POSTGRES_HOST", "localhost"),
PostgresPort:  getEnv("POSTGRES_PORT", "5432"),
PostgresUser:  getEnv("POSTGRES_USER", "auctioncore"),
PostgresPass:  getEnv("POSTGRES_PASSWORD", "auctioncore"),
PostgresDB:    getEnv("POSTGRES_DB", "auctioncore"),
PostgresSSL:   getEnv("POSTGRES_SSLMODE", "disable"),
RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
RedisPassword: getEnv("REDIS_PASSWORD", ""),
RedisDB:       getEnv("REDIS_DB", "0"),
}
}
