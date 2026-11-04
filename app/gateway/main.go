package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"jaxpay-go-backend/libs/config"
	"jaxpay-go-backend/libs/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title                       JaxPay API
// @version                     1.0
// @description                 Payment gateway terpusat untuk JaxLab, JAHRIS dan Qrupi.
// @BasePath                    /api/v1
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 "Bearer <token>" untuk admin.
// @securityDefinitions.apikey  ClientAPIKey
// @in                          header
// @name                        X-API-Key
// @description                 API key client. Wajib juga X-Timestamp dan X-Signature (HMAC-SHA256).
func main() {
	cfg := config.Get()
	if cfg.DatabaseURL == "" || cfg.JWTSecret == "" || cfg.EncryptionKey == "" {
		log.Fatal("POSTGRES_DSN_DEV, JWT_SECRET and ENCRYPTION_KEY are required")
	}
	db, e := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if e != nil {
		log.Fatalf("db: %v", e)
	}
	if e = db.AutoMigrate(models.AllModels()...); e != nil {
		log.Fatalf("migrate: %v", e)
	}
	if e = RegisterProviders(cfg); e != nil {
		log.Fatalf("providers: %v", e)
	}

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           NewRouter(db, cfg),
		ReadHeaderTimeout: 10 * time.Second,
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		log.Printf("JaxPay gateway listening on :%s", cfg.AppPort)
		if e := srv.ListenAndServe(); e != nil && !errors.Is(e, http.ErrServerClosed) {
			log.Fatal(e)
		}
	}()
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdown)
}
