package providerbanks

type DefaultFindDTO struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Sort     string `form:"sort"`
	Provider string `form:"provider"`
	BankCode string `form:"bank_code"`
	IsVA     *bool  `form:"is_va"`
	IsActive *bool  `form:"is_active"`
	Search   string `form:"search"`
}

type CreateDTO struct {
	Provider    string `json:"provider" binding:"required,max=50" example:"espay"`
	ProductCode string `json:"product_code" binding:"required,max=50" example:"BCAATM"`
	Name        string `json:"name" binding:"required,max=100" example:"BCA VA Online"`
	BankCode    string `json:"bank_code" binding:"required,numeric,len=3" example:"014"`
	IsVA        bool   `json:"is_va"`
	IsActive    *bool  `json:"is_active"`
}

type UpdateDTO struct {
	Name     *string `json:"name" binding:"omitempty,max=100"`
	BankCode *string `json:"bank_code" binding:"omitempty,numeric,len=3"`
	IsVA     *bool   `json:"is_va"`
	IsActive *bool   `json:"is_active"`
}
