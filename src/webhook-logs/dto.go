package webhooklogs

type DefaultFindDTO struct {
	Page      int    `form:"page" binding:"omitempty,min=1"`
	Limit     int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Sort      string `form:"sort"`
	PaymentID string `form:"payment_id" binding:"omitempty,uuid"`
	ClientID  string `form:"client_id" binding:"omitempty,uuid"`
	Event     string `form:"event"`
	Status    string `form:"status" binding:"omitempty,oneof=PENDING SUCCESS FAILED"`
}
