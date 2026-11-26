package espay

import "strconv"

type money float64

func (m money) MarshalJSON() ([]byte, error) {
	return []byte(strconv.FormatFloat(float64(m), 'f', 2, 64)), nil
}

type inquiryErrorBody struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
}

type inquiryBody struct {
	ResponseCode       string            `json:"responseCode"`
	ResponseMessage    string            `json:"responseMessage"`
	VirtualAccountData inquiryVADataFull `json:"virtualAccountData"`
}

type inquiryVADataFull struct {
	PartnerServiceID    string                `json:"partnerServiceId"`
	CustomerNo          string                `json:"customerNo"`
	VirtualAccountNo    string                `json:"virtualAccountNo"`
	VirtualAccountName  string                `json:"virtualAccountName"`
	VirtualAccountEmail string                `json:"virtualAccountEmail"`
	VirtualAccountPhone string                `json:"virtualAccountPhone"`
	InquiryRequestID    string                `json:"inquiryRequestId"`
	TotalAmount         amount                `json:"totalAmount"`
	BillDetails         []inquiryBillDetail   `json:"billDetails"`
	AdditionalInfo      inquiryAdditionalInfo `json:"additionalInfo"`
}

type inquiryBillDetail struct {
	BillDescription billDescription `json:"billDescription"`
}

type billDescription struct {
	English   string `json:"english"`
	Indonesia string `json:"indonesia"`
}

type inquiryAdditionalInfo struct {
	Token           string              `json:"token"`
	TransactionDate string              `json:"transactionDate"`
	DataMerchant    inquiryDataMerchant `json:"dataMerchant"`
	ShippingAddress inquiryShipping     `json:"shippingAddress"`
	Items           []inquiryItem       `json:"items"`
}

type inquiryDataMerchant struct {
	KodeCa         string          `json:"kodeCa"`
	KodeSubCa      string          `json:"kodeSubCa"`
	NoKontrak      string          `json:"noKontrak"`
	NamaPelanggan  string          `json:"namaPelanggan"`
	AngsuranKe     string          `json:"angsuranKe"`
	JmlBayarExcAdm money           `json:"jmlBayarExcAdm"`
	Denda          money           `json:"denda"`
	FeeCa          money           `json:"feeCa"`
	FeeSwitcher    money           `json:"feeSwitcher"`
	TotalAdmin     money           `json:"totalAdmin"`
	JumlahBayar    money           `json:"jumlahBayar"`
	MinimumAmount  money           `json:"minimumAmount"`
	TotalAngsuran  int             `json:"totalAngsuran"`
	Customer       inquiryCustomer `json:"customer"`
	JatuhTempo     string          `json:"jatuhTempo"`
	ListBills      []inquiryBill   `json:"listBills"`
}

type inquiryCustomer struct {
	Email string `json:"email"`
}

type inquiryBill struct {
	BillCode       int    `json:"billCode"`
	BillName       string `json:"billName"`
	BillAmount     amount `json:"billAmount"`
	BillSubCompany string `json:"billSubCompany"`
}

type inquiryShipping struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Address     string `json:"address"`
	City        string `json:"city"`
	PostalCode  string `json:"postalCode"`
	PhoneNumber string `json:"phoneNumber"`
	CountryCode string `json:"countryCode"`
}

type inquiryItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Price    string `json:"price"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Quantity string `json:"quantity"`
}
