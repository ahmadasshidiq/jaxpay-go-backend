package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"jaxpay-go-backend/libs/helpers"
	"jaxpay-go-backend/libs/models"
	"jaxpay-go-backend/libs/providers"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	errRouteNotFound   = errors.New("no active provider route for this method")
	errNotPending      = errors.New("payment is not pending")
	errRefConflict     = errors.New("reference_id already used with different method or amount")
	errProviderFailure = errors.New("provider request failed")
)

const defaultExpiredMinutes = 24 * 60

type PaymentService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *PaymentService {
	return &PaymentService{DB: db}
}

func (s *PaymentService) getAll(clientID uuid.UUID, d DefaultFindDTO) (*helpers.PaginatedResult, error) {
	q := s.DB.Model(&models.Payment{}).Where("payments.client_id = ?", clientID)
	filters := map[string]any{
		"payments.deleted_at.isnull": nil,
		"payments.status.eq":         d.Status,
		"payments.method.eq":         d.Method,
		"payments.reference_id.eq":   d.ReferenceID,
	}
	var rows []models.Payment
	r, e := helpers.BuildPaginatedQuery(q, d.Page, d.Limit, d.Sort, "payments.created_at DESC", filters, &rows)
	if e != nil {
		return nil, e
	}
	items := make([]PaymentResponse, len(rows))
	for i := range rows {
		items[i] = toResponse(&rows[i])
	}
	r.Items = items
	return r, nil
}

func (s *PaymentService) getByID(clientID, id uuid.UUID) (*models.Payment, error) {
	var p models.Payment
	e := s.DB.Where("id = ? AND client_id = ?", id, clientID).First(&p).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return &p, nil
}

func (s *PaymentService) findByReference(clientID uuid.UUID, ref string) (*models.Payment, error) {
	var p models.Payment
	e := s.DB.Unscoped().Where("client_id = ? AND reference_id = ?", clientID, ref).First(&p).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, e
}

// create bersifat idempotent per (client, reference_id). Return created=false jika payment sudah ada.
func (s *PaymentService) create(ctx context.Context, client *models.Client, d CreatePaymentDTO) (*models.Payment, bool, error) {
	if v, e := s.findByReference(client.ID, d.ReferenceID); e != nil || v != nil {
		return s.replay(v, d, e)
	}

	var route models.ProviderRoute
	e := s.DB.Where("client_id = ? AND method = ? AND is_active = ?", client.ID, d.Method, true).First(&route).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, false, errRouteNotFound
	}
	if e != nil {
		return nil, false, e
	}
	prov, e := providers.Get(route.Provider)
	if e != nil {
		return nil, false, errRouteNotFound
	}

	minutes := d.ExpiredMinutes
	if minutes == 0 {
		minutes = defaultExpiredMinutes
	}
	expiredAt := time.Now().Add(time.Duration(minutes) * time.Minute)
	p := &models.Payment{
		ClientID: client.ID, ReferenceID: d.ReferenceID, Method: d.Method, Provider: prov.Name(),
		Amount: d.Amount, Currency: "IDR", Status: models.StatusPending,
		CustomerName: d.CustomerName, CustomerEmail: d.CustomerEmail, Description: d.Description,
		ExpiredAt: &expiredAt,
	}
	if d.Metadata != nil {
		b, e := json.Marshal(d.Metadata)
		if e != nil {
			return nil, false, e
		}
		p.Metadata = datatypes.JSON(b)
	}

	// Insert dulu agar unique index (client_id, reference_id) mencegah dobel transaksi saat request paralel.
	if e = s.DB.Create(p).Error; e != nil {
		if v, e2 := s.findByReference(client.ID, d.ReferenceID); e2 == nil && v != nil {
			return s.replay(v, d, nil)
		}
		return nil, false, e
	}

	res, e := s.callProvider(ctx, prov, p)
	if e != nil {
		s.DB.Model(p).Update("status", models.StatusFailed)
		p.Status = models.StatusFailed
		return p, true, fmt.Errorf("%w: %v", errProviderFailure, e)
	}
	p.ProviderRef, p.VANumber, p.QRString = res.ProviderRef, res.VANumber, res.QRString
	if res.ExpiredAt != nil {
		p.ExpiredAt = res.ExpiredAt
	}
	e = s.DB.Model(p).Updates(map[string]any{
		"provider_ref": p.ProviderRef, "va_number": p.VANumber, "qr_string": p.QRString, "expired_at": p.ExpiredAt,
	}).Error
	return p, true, e
}

func (s *PaymentService) replay(v *models.Payment, d CreatePaymentDTO, e error) (*models.Payment, bool, error) {
	if e != nil {
		return nil, false, e
	}
	if v.Method != d.Method || v.Amount != d.Amount {
		return nil, false, errRefConflict
	}
	return v, false, nil
}

func (s *PaymentService) callProvider(ctx context.Context, prov providers.Provider, p *models.Payment) (*providers.PaymentResult, error) {
	switch p.Method {
	case "qris":
		return prov.CreateQRIS(ctx, providers.CreateQRISRequest{
			ReferenceID: p.ReferenceID, Amount: p.Amount, Description: p.Description, ExpiredAt: *p.ExpiredAt,
		})
	}
	return nil, providers.ErrNotSupported
}

func (s *PaymentService) cancel(ctx context.Context, clientID, id uuid.UUID) (*models.Payment, error) {
	p, e := s.getByID(clientID, id)
	if e != nil || p == nil {
		return p, e
	}
	if p.Status != models.StatusPending {
		return nil, errNotPending
	}
	prov, e := providers.Get(p.Provider)
	if e != nil {
		return nil, e
	}
	if e = prov.Cancel(ctx, p.ProviderRef); e != nil {
		return nil, fmt.Errorf("%w: %v", errProviderFailure, e)
	}
	if e = helpers.TransitionPayment(s.DB, p, models.StatusCancelled, nil); errors.Is(e, helpers.ErrStateConflict) {
		return nil, errNotPending
	}
	return p, e
}

func (s *PaymentService) checkStatus(ctx context.Context, clientID, id uuid.UUID) (*models.Payment, error) {
	p, e := s.getByID(clientID, id)
	if e != nil || p == nil || p.Status != models.StatusPending {
		return p, e
	}
	prov, e := providers.Get(p.Provider)
	if e != nil {
		return nil, e
	}
	st, e := prov.CheckStatus(ctx, p.ProviderRef)
	if errors.Is(e, providers.ErrNotSupported) {
		return p, nil
	}
	if e != nil {
		return nil, fmt.Errorf("%w: %v", errProviderFailure, e)
	}
	if st.Status == models.StatusPending || (st.Status == models.StatusPaid && st.Amount != 0 && st.Amount != p.Amount) {
		return p, nil
	}
	if e = helpers.TransitionPayment(s.DB, p, st.Status, st.PaidAt); errors.Is(e, helpers.ErrStateConflict) {
		return s.getByID(clientID, id)
	}
	return p, e
}
