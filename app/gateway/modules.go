package main

import (
	"log"

	"jaxpay-go-backend/libs/config"
	"jaxpay-go-backend/libs/models"
	"jaxpay-go-backend/libs/providers"
	"jaxpay-go-backend/libs/providers/espay"
	"jaxpay-go-backend/src/callbacks"
	"jaxpay-go-backend/src/clients"
	"jaxpay-go-backend/src/payments"
	providerbanks "jaxpay-go-backend/src/provider-banks"
	providerroutes "jaxpay-go-backend/src/provider-routes"
	webhooklogs "jaxpay-go-backend/src/webhook-logs"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterAllModules(router *gin.RouterGroup, db *gorm.DB) {
	clients.RegisterClientModule(router, db)
	providerroutes.RegisterProviderRouteModule(router, db)
	providerbanks.RegisterProviderBankModule(router, db)
	payments.RegisterPaymentModule(router, db)
	callbacks.RegisterCallbackModule(router, db)
	webhooklogs.RegisterWebhookLogModule(router, db)
}

func RegisterProviders(cfg *config.Config) error {
	if cfg.Espay.Enabled() {
		p, e := espay.New(cfg.Espay)
		if e != nil {
			return e
		}
		providers.Register(p)
		log.Println("provider registered: espay")
	}
	return nil
}

func SeedProviderBanks(db *gorm.DB) error {
	rows := make([]models.ProviderBank, 0, len(espay.Products()))
	for _, p := range espay.Products() {
		rows = append(rows, models.ProviderBank{Provider: "espay", ProductCode: p.Code, Name: p.Name, BankCode: p.BankCode, IsVA: p.IsVA, IsActive: true})
	}
	return providerbanks.Seed(db, "espay", rows)
}
