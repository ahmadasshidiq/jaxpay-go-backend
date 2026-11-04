package callbacks

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"jaxpay-go-backend/libs/helpers"
	"jaxpay-go-backend/libs/models"
	"jaxpay-go-backend/libs/providers"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var errUnknownProvider = errors.New("unknown provider")

type CallbackService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *CallbackService {
	return &CallbackService{DB: db}
}

// handle: simpan CallbackLog → verifikasi + parse → proses idempotent → balasan format provider.
func (s *CallbackService) handle(provider string, r *http.Request, body []byte) (int, any, error) {
	p, e := providers.Get(provider)
	if e != nil {
		return 0, nil, errUnknownProvider
	}
	log := &models.CallbackLog{Provider: provider, Headers: headersJSON(r), Payload: payloadJSON(body)}

	ev, e := p.ParseCallback(r, body)
	if ev == nil {
		ev = &providers.CallbackEvent{Provider: provider, Raw: body}
	}
	if e == nil {
		log.IsValid = true
		e = s.process(ev, log)
	}
	if e != nil {
		log.Error = e.Error()
	}
	if e2 := s.DB.Create(log).Error; e2 != nil {
		return 0, nil, e2
	}
	code, v := p.CallbackResponse(ev, e)
	return code, v, nil
}

func (s *CallbackService) process(ev *providers.CallbackEvent, log *models.CallbackLog) error {
	p, e := s.findPayment(ev)
	if e != nil {
		return e
	}
	if p == nil {
		return providers.ErrPaymentNotFound
	}
	log.PaymentID = &p.ID
	expired := p.ExpiredAt != nil && time.Now().After(*p.ExpiredAt)

	if ev.Type == providers.CallbackInquiry {
		switch {
		case p.Status == models.StatusPaid:
			return providers.ErrAlreadyPaid
		case p.Status != models.StatusPending || expired:
			return providers.ErrPaymentExpired
		}
		ev.Bill = &providers.Bill{
			CustomerName: p.CustomerName, CustomerEmail: p.CustomerEmail, Description: p.Description,
			Amount: p.Amount, CreatedAt: p.CreatedAt, ExpiredAt: p.ExpiredAt,
		}
		return nil
	}

	if p.Status == models.StatusPaid {
		return nil // idempotent: callback ulang untuk payment yang sudah PAID
	}
	if ev.Status == models.StatusPending {
		return nil
	}
	if p.Status != models.StatusPending || (expired && ev.Status == models.StatusPaid) {
		return providers.ErrPaymentExpired
	}
	if ev.Status == models.StatusPaid && ev.Amount != p.Amount {
		return providers.ErrAmountMismatch
	}
	paidAt := ev.PaidAt
	if ev.Status == models.StatusPaid && paidAt == nil {
		now := time.Now()
		paidAt = &now
	}
	e = helpers.TransitionPayment(s.DB, p, ev.Status, paidAt)
	if errors.Is(e, helpers.ErrStateConflict) {
		// Diproses paralel oleh callback lain / worker.
		if s.DB.First(p, "id = ?", p.ID).Error == nil && p.Status == ev.Status {
			return nil
		}
		return providers.ErrPaymentExpired
	}
	return e
}

// findPayment mencari payment via ProviderRef/VANumber, mendahulukan yang PENDING (nomor VA bisa dipakai ulang).
func (s *CallbackService) findPayment(ev *providers.CallbackEvent) (*models.Payment, error) {
	q := s.DB.Where("provider = ?", ev.Provider)
	switch {
	case ev.VANumber != "":
		q = q.Where("va_number = ?", ev.VANumber)
	case ev.ProviderRef != "":
		q = q.Where("provider_ref = ?", ev.ProviderRef)
	default:
		return nil, nil
	}
	var p models.Payment
	e := q.Order(gorm.Expr("CASE WHEN status = ? THEN 0 WHEN status = ? THEN 1 ELSE 2 END, created_at DESC",
		models.StatusPending, models.StatusPaid)).First(&p).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return &p, nil
}

func headersJSON(r *http.Request) datatypes.JSON {
	h := map[string]string{}
	for k, v := range r.Header {
		if k == "Authorization" || len(v) == 0 {
			continue // jangan simpan token
		}
		h[k] = v[0]
	}
	h["X-Remote-Addr"] = r.RemoteAddr
	b, _ := json.Marshal(h)
	return datatypes.JSON(b)
}

func payloadJSON(body []byte) datatypes.JSON {
	if json.Valid(body) {
		return datatypes.JSON(body)
	}
	b, _ := json.Marshal(string(body))
	return datatypes.JSON(b)
}
