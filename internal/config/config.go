package config

import "os"

type Config struct {
	ServiceName string
	Port        string
	PostgresDSN string
	RedisAddr   string
}

func Load(serviceName, defaultPort string) Config {
	return Config{
		ServiceName: serviceName,
		Port:        getEnv("PORT", defaultPort),
		PostgresDSN: getEnv("POSTGRES_DSN", "postgres://kodrapay:kodrapay@postgres:5432/kodrapay?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "redis:6379"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
