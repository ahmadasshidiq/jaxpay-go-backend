package config

import (
	"os"
	"strconv"
	"sync"

	"github.com/joho/godotenv"
)

type EspayConfig struct {
	BaseURL        string
	PartnerID      string
	MerchantID     string
	ChannelID      string
	PrivateKeyPath string // private key merchant
	PublicKeyPath  string // public key Espay
}

func (c EspayConfig) Enabled() bool { return c.BaseURL != "" && c.PartnerID != "" }

type Config struct {
	AppPort            string
	DatabaseURL        string
	JWTSecret          string
	RedisURL           string
	EncryptionKey      string
	RateLimitPerMinute int
	Espay              EspayConfig
}

var (
	cfg  *Config
	once sync.Once
)

// Get loads the config once from env (and .env if present).
func Get() *Config {
	once.Do(func() {
		_ = godotenv.Load()
		cfg = &Config{
			AppPort:            env("APP_PORT", "8080"),
			DatabaseURL:        env("POSTGRES_DSN_DEV", ""),
			JWTSecret:          env("JWT_SECRET", ""),
			RedisURL:           env("REDIS_URL", ""),
			EncryptionKey:      env("ENCRYPTION_KEY", ""),
			RateLimitPerMinute: envInt("RATE_LIMIT_PER_MINUTE", 120),
			Espay: EspayConfig{
				BaseURL:        env("ESPAY_BASE_URL", ""),
				PartnerID:      env("ESPAY_PARTNER_ID", ""),
				MerchantID:     env("ESPAY_MERCHANT_ID", ""),
				ChannelID:      env("ESPAY_CHANNEL_ID", ""),
				PrivateKeyPath: env("ESPAY_PRIVATE_KEY_PATH", ""),
				PublicKeyPath:  env("ESPAY_PUBLIC_KEY_PATH", ""),
			},
		}
	})
	return cfg
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v, e := strconv.Atoi(os.Getenv(k)); e == nil {
		return v
	}
	return def
}
