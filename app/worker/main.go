package main

import (
	"context"
	"log"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"jaxpay-go-backend/libs/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	cfg := config.Get()
	if cfg.DatabaseURL == "" || cfg.EncryptionKey == "" {
		log.Fatal("POSTGRES_DSN_DEV and ENCRYPTION_KEY are required")
	}
	db, e := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if e != nil {
		log.Fatalf("db: %v", e)
	}
	if e = RegisterProviders(cfg); e != nil {
		log.Fatalf("providers: %v", e)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup
	for _, j := range AllJobs() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			log.Printf("job started: %s (every %s)", j.Name, j.Interval)
			t := time.NewTicker(j.Interval)
			defer t.Stop()
			for {
				if e := j.Run(ctx, db); e != nil {
					log.Printf("job %s: %v", j.Name, e)
				}
				select {
				case <-ctx.Done():
					return
				case <-t.C:
				}
			}
		}()
	}
	wg.Wait()
}
