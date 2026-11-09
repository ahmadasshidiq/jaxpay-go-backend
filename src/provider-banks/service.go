package providerbanks

import (
	"errors"

	"jaxpay-go-backend/libs/helpers"
	"jaxpay-go-backend/libs/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProviderBankService struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *ProviderBankService {
	return &ProviderBankService{DB: db}
}

func Seed(db *gorm.DB, provider string, rows []models.ProviderBank) error {
	var n int64
	if e := db.Unscoped().Model(&models.ProviderBank{}).Where("provider = ?", provider).Count(&n).Error; e != nil || n > 0 {
		return e
	}
	return db.Create(&rows).Error
}

func (s *ProviderBankService) getAll(d DefaultFindDTO) (*helpers.PaginatedResult, error) {
	filters := map[string]any{
		"provider_banks.deleted_at.isnull": nil,
		"provider_banks.provider.eq":       d.Provider,
		"provider_banks.bank_code.eq":      d.BankCode,
		"provider_banks.name.ilike":        d.Search,
	}
	if d.IsVA != nil {
		filters["provider_banks.is_va.eq"] = *d.IsVA
	}
	if d.IsActive != nil {
		filters["provider_banks.is_active.eq"] = *d.IsActive
	}
	var v []models.ProviderBank
	return helpers.BuildPaginatedQuery(s.DB.Model(&models.ProviderBank{}), d.Page, d.Limit, d.Sort,
		"provider_banks.name ASC", filters, &v)
}

func (s *ProviderBankService) getByID(id uuid.UUID) (*models.ProviderBank, error) {
	var v models.ProviderBank
	e := s.DB.First(&v, "id = ?", id).Error
	if errors.Is(e, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return &v, nil
}

func (s *ProviderBankService) create(d CreateDTO) (*models.ProviderBank, error) {
	v := models.ProviderBank{
		Provider: d.Provider, ProductCode: d.ProductCode, Name: d.Name, BankCode: d.BankCode, IsVA: d.IsVA, IsActive: true,
	}
	if e := s.DB.Create(&v).Error; e != nil {
		return nil, e
	}
	if d.IsActive != nil && !*d.IsActive {
		s.DB.Model(&v).Update("is_active", false)
	}
	return &v, nil
}

func (s *ProviderBankService) update(id uuid.UUID, d UpdateDTO) (*models.ProviderBank, error) {
	v, e := s.getByID(id)
	if e != nil || v == nil {
		return v, e
	}
	updates := map[string]any{}
	if d.Name != nil {
		updates["name"] = *d.Name
	}
	if d.BankCode != nil {
		updates["bank_code"] = *d.BankCode
	}
	if d.IsVA != nil {
		updates["is_va"] = *d.IsVA
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

func (s *ProviderBankService) archive(id uuid.UUID) (bool, error) {
	return s.delete(id)
}

func (s *ProviderBankService) delete(id uuid.UUID) (bool, error) {
	r := s.DB.Unscoped().Delete(&models.ProviderBank{}, "id = ?", id)
	return r.RowsAffected > 0, r.Error
}
