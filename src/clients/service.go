package clients

import (
	"errors"

	"jaxpay-go-backend/libs/cryptography"
	"jaxpay-go-backend/libs/helpers"
	"jaxpay-go-backend/libs/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ClientService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *ClientService {
	return &ClientService{DB: db}
}

func (s *ClientService) getAll(d DefaultFindDTO) (*helpers.PaginatedResult, error) {
	q := s.DB.Model(&models.Client{})
	if d.Search != "" {
		q = q.Where("clients.code ILIKE ? OR clients.name ILIKE ?", "%"+d.Search+"%", "%"+d.Search+"%")
	}
	filters := map[string]any{
		"clients.deleted_at.isnull": nil,
		"clients.is_active.eq":      d.IsActive,
	}
	var v []models.Client
	return helpers.BuildPaginatedQuery(q, d.Page, d.Limit, d.Sort, "clients.created_at DESC", filters, &v)
}

func (s *ClientService) getByID(id uuid.UUID) (*models.Client, error) {
	var v models.Client
	e := s.DB.First(&v, "id = ?", id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return &v, nil
}

func newCredentials() (apiKey, apiSecret, webhookSecret, encAPI, encWebhook string, e error) {
	apiKey = "jxp_" + cryptography.RandomToken(16)
	apiSecret = cryptography.RandomToken(32)
	webhookSecret = "whsec_" + cryptography.RandomToken(32)
	if encAPI, e = cryptography.Encrypt(apiSecret); e != nil {
		return
	}
	encWebhook, e = cryptography.Encrypt(webhookSecret)
	return
}

func (s *ClientService) create(d CreateDTO) (*CredentialsDTO, error) {
	key, secret, whsec, encAPI, encWh, e := newCredentials()
	if e != nil {
		return nil, e
	}
	v := models.Client{
		Code: d.Code, Name: d.Name, CallbackURL: d.CallbackURL, VAPrefix: d.VAPrefix,
		APIKey: key, APISecretHash: encAPI, WebhookSecret: encWh, IsActive: true,
	}
	if e = s.DB.Create(&v).Error; e != nil {
		return nil, e
	}
	if d.IsActive != nil && !*d.IsActive {
		s.DB.Model(&v).Update("is_active", false)
	}
	return &CredentialsDTO{ID: v.ID, APIKey: key, APISecret: secret, WebhookSecret: whsec}, nil
}

func (s *ClientService) update(id uuid.UUID, d UpdateDTO) (*models.Client, error) {
	v, e := s.getByID(id)
	if e != nil || v == nil {
		return v, e
	}
	updates := map[string]any{}
	if d.Name != nil {
		updates["name"] = *d.Name
	}
	if d.CallbackURL != nil {
		updates["callback_url"] = *d.CallbackURL
	}
	if d.VAPrefix != nil {
		updates["va_prefix"] = *d.VAPrefix
	}
	if d.IsActive != nil {
		updates["is_active"] = *d.IsActive
	}
	if len(updates) > 0 {
		if e = s.DB.Model(v).Updates(updates).Error; e != nil {
			return nil, e
		}
	}
	return s.getByID(id)
}

func (s *ClientService) rotateSecret(id uuid.UUID) (*CredentialsDTO, error) {
	v, e := s.getByID(id)
	if e != nil || v == nil {
		return nil, e
	}
	key, secret, whsec, encAPI, encWh, e := newCredentials()
	if e != nil {
		return nil, e
	}
	e = s.DB.Model(v).Updates(map[string]any{"api_key": key, "api_secret_hash": encAPI, "webhook_secret": encWh}).Error
	if e != nil {
		return nil, e
	}
	return &CredentialsDTO{ID: v.ID, APIKey: key, APISecret: secret, WebhookSecret: whsec}, nil
}

func (s *ClientService) archive(id uuid.UUID) (bool, error) {
	r := s.DB.Delete(&models.Client{}, "id = ?", id)
	return r.RowsAffected > 0, r.Error
}

func (s *ClientService) delete(id uuid.UUID) (bool, error) {
	r := s.DB.Unscoped().Delete(&models.Client{}, "id = ?", id)
	return r.RowsAffected > 0, r.Error
}
