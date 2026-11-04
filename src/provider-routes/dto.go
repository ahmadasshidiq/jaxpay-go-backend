package providerroutes

type DefaultFindDTO struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Sort     string `form:"sort"`
	ClientID string `form:"client_id" binding:"omitempty,uuid"`
	Method   string `form:"method"`
	Provider string `form:"provider"`
}

type CreateDTO struct {
	ClientID string `json:"client_id" binding:"required,uuid"`
	Method   string `json:"method" binding:"required,oneof=qris"`
	Provider string `json:"provider" binding:"required,max=50"`
	IsActive *bool  `json:"is_active"`
}

type UpdateDTO struct {
	Provider *string `json:"provider" binding:"omitempty,max=50"`
	IsActive *bool   `json:"is_active"`
}
