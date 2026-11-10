package config

import (
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

type EspayConfig struct {
	BaseURL        string
	PartnerID      string
	MerchantID     string
	ChannelID      string
	PrivateKeyPath string
	PublicKeyPath  string
	Password       string
	SignatureKey   string
	MerchantPGURL  string
}

func (c EspayConfig) Enabled() bool { return c.BaseURL != "" && c.PartnerID != "" }

type CORSConfig struct {
	AllowOrigins []string
	AllowMethods []string
	AllowHeaders []string
}

const (
	defaultCORSMethods = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	defaultCORSHeaders = "Authorization, Content-Type, X-API-Key, X-Timestamp, X-Signature"
)

type Config struct {
	AppEnv             string
	AppPort            string
	DatabaseURL        string
	JWTSecret          string
	RedisURL           string
	EncryptionKey      string
	RateLimitPerMinute int
	Espay              EspayConfig
	CORS               CORSConfig
}

var (
	cfg  *Config
	once sync.Once
)

func Get() *Config {
	once.Do(func() {
		_ = godotenv.Load()
		cfg = &Config{
			AppEnv:             env("APP_ENV", "development"),
			AppPort:            env("APP_PORT", "8080"),
			DatabaseURL:        env("POSTGRES_DSN_DEV", ""),
			JWTSecret:          env("JWT_SECRET", ""),
			RedisURL:           env("REDIS_URL", ""),
			EncryptionKey:      env("ENCRYPTION_KEY", ""),
			RateLimitPerMinute: envInt("RATE_LIMIT_PER_MINUTE", 120),
			CORS: CORSConfig{
				AllowOrigins: envList("CORS_ALLOW_ORIGINS", "*"),
				AllowMethods: envList("CORS_ALLOW_METHODS", defaultCORSMethods),
				AllowHeaders: append(splitList(defaultCORSHeaders), envList("CORS_EXTRA_HEADERS", "")...),
			},
			Espay: EspayConfig{
				BaseURL:        env("ESPAY_BASE_URL", ""),
				PartnerID:      env("ESPAY_PARTNER_ID", ""),
				MerchantID:     env("ESPAY_MERCHANT_ID", ""),
				ChannelID:      env("ESPAY_CHANNEL_ID", ""),
				PrivateKeyPath: env("ESPAY_PRIVATE_KEY_PATH", ""),
				PublicKeyPath:  env("ESPAY_PUBLIC_KEY_PATH", ""),
				Password:       env("ESPAY_PASSWORD", ""),
				SignatureKey:   env("ESPAY_SIGNATURE_KEY", ""),
				MerchantPGURL:  env("ESPAY_MERCHANTPG_URL", ""),
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

func envList(k, def string) []string { return splitList(env(k, def)) }

func splitList(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
