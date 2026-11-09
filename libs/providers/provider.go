package providers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"jaxpay-go-backend/libs/models"
)

type Provider interface {
	Name() string
	CreateVA(ctx context.Context, req CreateVARequest) (*PaymentResult, error)
	CreateQRIS(ctx context.Context, req CreateQRISRequest) (*PaymentResult, error)
	CreateDirectDebit(ctx context.Context, req CreateDirectDebitRequest) (*PaymentResult, error)
	CheckStatus(ctx context.Context, method, providerRef string) (*StatusResult, error)
	Cancel(ctx context.Context, method, providerRef string) error
	ParseCallback(r *http.Request, body []byte) (*CallbackEvent, error)
	CallbackResponse(ev *CallbackEvent, err error) (int, any)
}

func NewProviderRef() string {
	b := make([]byte, 10)
	_, _ = rand.Read(b)
	return "JXP" + strings.ToUpper(hex.EncodeToString(b))
}

var (
	ErrNotSupported     = errors.New("method not supported by provider")
	ErrInvalidSignature = errors.New("invalid signature")
	ErrInvalidRequest   = errors.New("invalid request")
	ErrPaymentNotFound  = errors.New("payment not found")
	ErrAlreadyPaid      = errors.New("payment already paid")
	ErrPaymentExpired   = errors.New("payment expired or closed")
	ErrAmountMismatch   = errors.New("amount mismatch")
)

type CreateVARequest struct {
	ProviderRef   string
	ReferenceID   string
	Amount        int64
	CustomerName  string
	CustomerEmail string
	CustomerPhone string
	Description   string
	BankCode      string
	VAPrefix      string
	ExpiredAt     time.Time
}

type CreateDirectDebitRequest struct {
	ProviderRef   string
	Amount        int64
	CustomerName  string
	CustomerEmail string
	CustomerPhone string
	BankCode      string
	ProductCode   string
	ReturnURL     string
	ExpiredAt     time.Time
}

type CreateQRISRequest struct {
	ProviderRef string
	ReferenceID string
	Amount      int64
	Description string
	ExpiredAt   time.Time
}

type PaymentResult struct {
	ProviderRef  string
	VANumber     *string
	QRString     *string
	QRURL        *string
	RedirectURL  *string
	ExpiredAt    *time.Time
	ProviderData []byte
}

type StatusResult struct {
	ProviderRef string
	Status      models.PaymentStatus
	Amount      int64
	PaidAt      *time.Time
}

const (
	CallbackInquiry    = "inquiry"
	CallbackPayment    = "payment"
	CallbackSettlement = "settlement"
)

type CallbackEvent struct {
	Provider    string
	Type        string
	Endpoint    string
	ProviderRef string
	VANumber    string
	Amount      int64
	Status      models.PaymentStatus
	PaidAt      *time.Time
	Raw         []byte
	Bill        *Bill
}

type Bill struct {
	CustomerName  string
	CustomerEmail string
	Description   string
	Amount        int64
	CreatedAt     time.Time
	ExpiredAt     *time.Time
}

var (
	mu       sync.RWMutex
	registry = map[string]Provider{}
)

func Register(p Provider) {
	mu.Lock()
	defer mu.Unlock()
	registry[p.Name()] = p
}

func Get(name string) (Provider, error) {
	mu.RLock()
	defer mu.RUnlock()
	if p, ok := registry[name]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("provider %q not registered", name)
}
