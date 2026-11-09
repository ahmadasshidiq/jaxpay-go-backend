package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"jaxpay-go-backend/libs/config"
	"jaxpay-go-backend/libs/cryptography"
	"jaxpay-go-backend/libs/helpers"
	"jaxpay-go-backend/libs/models"
	"jaxpay-go-backend/libs/providers"
	"jaxpay-go-backend/libs/providers/espay"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Job struct {
	Name     string
	Interval time.Duration
	Run      func(ctx context.Context, db *gorm.DB) error
}

func AllJobs() []Job {
	return []Job{
		{Name: "webhook-dispatch", Interval: 10 * time.Second, Run: dispatchWebhooks},
		{Name: "expire-payments", Interval: time.Minute, Run: expirePayments},
		{Name: "reconcile-payments", Interval: 2 * time.Minute, Run: reconcilePayments},
	}
}

func RegisterProviders(cfg *config.Config) error {
	if cfg.Espay.Enabled() {
		p, e := espay.New(cfg.Espay)
		if e != nil {
			return e
		}
		providers.Register(p)
	}
	return nil
}

const (
	maxWebhookAttempts = 8
	webhookLease       = 2 * time.Minute
	reconcileAfter     = 10 * time.Minute
)

var backoff = []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 6 * time.Hour}

var webhookHTTP = &http.Client{Timeout: 10 * time.Second}

func dispatchWebhooks(ctx context.Context, db *gorm.DB) error {
	var batch []models.WebhookLog
	e := db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND next_retry_at <= ?", models.WebhookPending, time.Now()).
			Order("next_retry_at").Limit(50).Find(&batch).Error; e != nil {
			return e
		}
		if len(batch) == 0 {
			return nil
		}
		ids := make([]any, len(batch))
		for i := range batch {
			ids[i] = batch[i].ID
		}
		return tx.Model(&models.WebhookLog{}).Where("id IN ?", ids).Update("next_retry_at", time.Now().Add(webhookLease)).Error
	})
	if e != nil {
		return e
	}
	for i := range batch {
		if ctx.Err() != nil {
			return nil
		}
		deliver(ctx, db, &batch[i])
	}
	return nil
}

func deliver(ctx context.Context, db *gorm.DB, w *models.WebhookLog) {
	code, body, e := send(ctx, db, w)
	attempt := w.Attempt + 1
	updates := map[string]any{"attempt": attempt, "response_code": code, "response_body": body}
	switch {
	case e == nil && code >= 200 && code < 300:
		updates["status"] = models.WebhookSuccess
		updates["next_retry_at"] = nil
	case attempt >= maxWebhookAttempts:
		updates["status"] = models.WebhookFailed
		updates["next_retry_at"] = nil
	default:
		updates["next_retry_at"] = time.Now().Add(backoff[min(attempt-1, len(backoff)-1)])
	}
	if e != nil {
		updates["response_body"] = e.Error()
	}
	if e2 := db.Model(w).Updates(updates).Error; e2 != nil {
		log.Printf("webhook %s: save result: %v", w.ID, e2)
	}
}

func send(ctx context.Context, db *gorm.DB, w *models.WebhookLog) (int, string, error) {
	var c models.Client
	if e := db.Unscoped().First(&c, "id = ?", w.ClientID).Error; e != nil {
		return 0, "", e
	}
	if c.CallbackURL == "" {
		return 0, "", errors.New("client has no callback_url")
	}
	secret, e := cryptography.Decrypt(c.WebhookSecret)
	if e != nil {
		return 0, "", errors.New("cannot decrypt webhook secret")
	}
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, c.CallbackURL, bytes.NewReader(w.Payload))
	if e != nil {
		return 0, "", e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-JaxPay-Event", w.Event)
	req.Header.Set("X-JaxPay-Timestamp", ts)
	req.Header.Set("X-JaxPay-Signature", cryptography.HMACSHA256Hex(secret, ts+"."+string(w.Payload)))
	res, e := webhookHTTP.Do(req)
	if e != nil {
		return 0, "", e
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	return res.StatusCode, string(b), nil
}

func expirePayments(ctx context.Context, db *gorm.DB) error {
	var rows []models.Payment
	if e := db.Where("status = ? AND expired_at < ?", models.StatusPending, time.Now()).
		Limit(200).Find(&rows).Error; e != nil {
		return e
	}
	for i := range rows {
		p := &rows[i]
		if paid := syncFromProvider(ctx, db, p); paid {
			continue
		}
		if prov, e := providers.Get(p.Provider); e == nil && p.ProviderRef != "" {
			_ = prov.Cancel(ctx, p.Method, p.ProviderRef)
		}
		if e := helpers.TransitionPayment(db, p, models.StatusExpired, nil); e != nil && !errors.Is(e, helpers.ErrStateConflict) {
			log.Printf("expire %s: %v", p.ID, e)
		}
	}
	return nil
}

func reconcilePayments(ctx context.Context, db *gorm.DB) error {
	var rows []models.Payment
	now := time.Now()
	if e := db.Where("status = ? AND provider_ref <> '' AND created_at < ? AND expired_at > ? AND updated_at < ?",
		models.StatusPending, now.Add(-reconcileAfter), now, now.Add(-5*time.Minute)).
		Order("updated_at").Limit(100).Find(&rows).Error; e != nil {
		return e
	}
	for i := range rows {
		if ctx.Err() != nil {
			return nil
		}
		p := &rows[i]
		if !syncFromProvider(ctx, db, p) {
			db.Model(p).UpdateColumn("updated_at", time.Now())
		}
	}
	return nil
}

func syncFromProvider(ctx context.Context, db *gorm.DB, p *models.Payment) bool {
	prov, e := providers.Get(p.Provider)
	if e != nil || p.ProviderRef == "" {
		return false
	}
	st, e := prov.CheckStatus(ctx, p.Method, p.ProviderRef)
	if e != nil {
		if !errors.Is(e, providers.ErrNotSupported) {
			log.Printf("check status %s: %v", p.ID, e)
		}
		return false
	}
	if st.Status == models.StatusPending {
		return false
	}
	if st.Status == models.StatusPaid && st.Amount != 0 && st.Amount != p.Amount {
		log.Printf("check status %s: amount mismatch (provider %d, expected %d)", p.ID, st.Amount, p.Amount)
		return false
	}
	paidAt := st.PaidAt
	if st.Status == models.StatusPaid && paidAt == nil {
		paidAt = &time.Time{}
		*paidAt = time.Now()
	}
	e = helpers.TransitionPayment(db, p, st.Status, paidAt)
	return e == nil || errors.Is(e, helpers.ErrStateConflict)
}
