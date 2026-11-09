package providerroutes

import (
	"errors"

	"jaxpay-go-backend/libs/helpers"
	"jaxpay-go-backend/libs/models"
	"jaxpay-go-backend/libs/providers"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var errClientNotFound = errors.New("client not found")

type ProviderRouteService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *ProviderRouteService {
	return &ProviderRouteService{DB: db}
}

func (s *ProviderRouteService) getAll(d DefaultFindDTO) (*helpers.PaginatedResult, error) {
	filters := map[string]any{
		"provider_routes.deleted_at.isnull": nil,
		"provider_routes.client_id.eq":      d.ClientID,
		"provider_routes.method.eq":         d.Method,
		"provider_routes.provider.eq":       d.Provider,
	}
	var v []models.ProviderRoute
	return helpers.BuildPaginatedQuery(s.DB.Model(&models.ProviderRoute{}), d.Page, d.Limit, d.Sort,
		"provider_routes.created_at DESC", filters, &v)
}

func (s *ProviderRouteService) getByID(id uuid.UUID) (*models.ProviderRoute, error) {
	var v models.ProviderRoute
	e := s.DB.First(&v, "id = ?", id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return &v, nil
}

func (s *ProviderRouteService) create(d CreateDTO) (*models.ProviderRoute, error) {
	if _, e := providers.Get(d.Provider); e != nil {
		return nil, e
	}
	clientID := uuid.MustParse(d.ClientID)
	var n int64
	if e := s.DB.Model(&models.Client{}).Where("id = ?", clientID).Count(&n).Error; e != nil {
		return nil, e
	}
	if n == 0 {
		return nil, errClientNotFound
	}
	v := models.ProviderRoute{ClientID: clientID, Method: d.Method, Provider: d.Provider, IsActive: true}
	if e := s.DB.Create(&v).Error; e != nil {
		return nil, e
	}
	if d.IsActive != nil && !*d.IsActive {
		s.DB.Model(&v).Update("is_active", false)
	}
	return &v, nil
}

func (s *ProviderRouteService) update(id uuid.UUID, d UpdateDTO) (*models.ProviderRoute, error) {
	v, e := s.getByID(id)
	if e != nil || v == nil {
		return v, e
	}
	updates := map[string]any{}
	if d.Provider != nil {
		if _, e = providers.Get(*d.Provider); e != nil {
			return nil, e
		}
		updates["provider"] = *d.Provider
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

func (s *ProviderRouteService) archive(id uuid.UUID) (bool, error) {
	return s.delete(id)
}

func (s *ProviderRouteService) delete(id uuid.UUID) (bool, error) {
	r := s.DB.Unscoped().Delete(&models.ProviderRoute{}, "id = ?", id)
	return r.RowsAffected > 0, r.Error
}
