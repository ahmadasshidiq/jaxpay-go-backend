# JaxPay — Payment Gateway Service (Golang)

> Dokumen ini adalah **spesifikasi arsitektur + prompt** untuk membangun JaxPay.
> Bisa langsung diberikan ke AI coding assistant sebagai konteks project.

---

## 1. Tujuan

JaxPay adalah service pembayaran terpusat yang dipakai bersama oleh beberapa aplikasi internal (**JaxLab, JAHRIS, Qrupi**). Aplikasi tidak lagi berbicara langsung dengan bank/payment provider — semua lewat JaxPay.

```
JaxLab ─┐
JAHRIS ─┼──► JaxPay (Go) ──► Provider (BCA, Espay, Midtrans, ...)
Qrupi ──┘        │                     │
                 │◄──── callback ──────┘
                 └────► webhook ke callback_url tiap app
```

**Fase 1:** BCA — Payment Host to Host (VA) + QRIS (QR-MPM).
**Fase berikutnya:** Espay (QRIS), Midtrans (VA/QRIS), provider lain — cukup tambah adapter.

---

## 2. Tech Stack

- Go + **Gin** (HTTP)
- **GORM** + PostgreSQL
- **Swaggo** (anotasi `@Summary`, `@Router`, dst)
- `google/uuid` untuk primary key
- Redis (opsional) untuk queue retry webhook — atau tabel outbox

Module name: `jaxpay-backend`

---

## 3. Struktur Folder (WAJIB mengikuti pola ini)

```
jaxpay-backend/
├── app/
│   ├── gateway/                 # entrypoint HTTP utama
│   │   ├── main.go
│   │   ├── modules.go           # RegisterAllModules
│   │   ├── router.go
│   │   ├── cors.go
│   │   └── rate_limiter.go
│   └── worker/                  # entrypoint worker (retry webhook, expire, reconcile)
│       ├── main.go
│       └── modules.go
├── docs/                        # hasil swag init
├── libs/
│   ├── cryptography/            # JWTMiddleware, AccessMiddleware, ClientAuthMiddleware, HMAC
│   ├── helpers/                 # RespondSuccess, RespondError, HandleRemove, BuildPaginatedQuery
│   ├── models/                  # semua GORM model
│   ├── config/                  # load env
│   └── providers/               # adapter payment provider
│       ├── provider.go          # interface + tipe umum + registry
│       ├── bca/
│       │   ├── client.go        # auth token, signature SNAP
│       │   ├── va.go
│       │   ├── qris.go
│       │   └── callback.go
│       ├── espay/
│       └── midtrans/
└── src/
    ├── clients/                 # CRUD client app (jaxlab, jahris, qrupi) — admin
    ├── provider-routes/         # CRUD mapping client+method → provider — admin
    ├── payments/                # API untuk app: create, get, cancel, list
    ├── callbacks/               # endpoint callback dari provider (public, signature-verified)
    └── webhook-logs/            # riwayat pengiriman webhook ke app — admin
```

Setiap folder di `src/` berisi **tepat 4 file**:

| File | Isi |
|---|---|
| `controller.go` | struct `XxxController{Service *XxxService}`, `NewController`, handler Gin + anotasi Swagger |
| `dto.go` | `DefaultFindDTO`, `CreateDTO`, `UpdateDTO` (+ DTO khusus) dengan tag `binding` |
| `module.go` | `RegisterXxxModule(router *gin.RouterGroup, db *gorm.DB)` — routing + middleware |
| `service.go` | struct `XxxService{DB *gorm.DB}`, `NewService`, method **lowercase** (unexported) |

---

## 4. Konvensi Kode

1. **Controller**
   - Variabel pendek: `x *gin.Context`, `d` untuk DTO, `v` untuk hasil, `e` untuk error.
   - Bind: `x.ShouldBindJSON(&d)` / `x.ShouldBindQuery(&d)` → gagal = `RespondError(x, "<module>", 400, e)`.
   - Service error = 500, hasil `nil` = 404.
   - Sukses create → `RespondSuccess(x, "<module>", 201, v.ID)`.
   - Archive/Delete pakai `helpers.HandleRemove`.
   - Setiap handler wajib anotasi Swagger lengkap (`@Summary`, `@Tags`, `@Param`, `@Security`, `@Router`).

2. **Service**
   - Method unexported: `getAll`, `getByID`, `create`, `update`, `archive`, `delete`.
   - `getByID` mengembalikan `nil, nil` saat `gorm.ErrRecordNotFound`.
   - List pakai `helpers.BuildPaginatedQuery` + filter `<table>.deleted_at.isnull`.
   - Update pakai pointer field di `UpdateDTO`, hanya set field yang `!= nil`.
   - Scoping per client: ambil `ctx.GetString("client_id")` (analog `institution_id`).

3. **Module**
   - Admin route: `cryptography.JWTMiddleware(db)` + `AccessMiddleware.Handler("<module>", "<action>")`.
   - Route untuk app client: `cryptography.ClientAuthMiddleware(db)` (API key + HMAC signature).
   - Route callback provider: **tanpa JWT**, verifikasi signature di adapter provider.

4. **modules.go** — daftarkan semua module:
```go
func RegisterAllModules(router *gin.RouterGroup, db *gorm.DB) {
	clients.RegisterClientModule(router, db)
	providerroutes.RegisterProviderRouteModule(router, db)
	payments.RegisterPaymentModule(router, db)
	callbacks.RegisterCallbackModule(router, db)
	webhooklogs.RegisterWebhookLogModule(router, db)
}
```

---

## 5. Model Database (`libs/models`)

```go
// Client = aplikasi yang memakai JaxPay (jaxlab, jahris, qrupi)
type Client struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Code          string    `gorm:"uniqueIndex;size:50"`   // "jaxlab", "jahris", "qrupi"
	Name          string    `gorm:"size:255"`
	APIKey        string    `gorm:"uniqueIndex;size:100"`
	APISecretHash string    `gorm:"size:255"`              // untuk verifikasi HMAC request
	WebhookSecret string    `gorm:"size:255"`              // untuk menandatangani webhook keluar
	CallbackURL   string    `gorm:"size:500"`
	VAPrefix      string    `gorm:"size:20"`               // pemetaan VA → client
	IsActive      bool      `gorm:"default:true"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

// ProviderRoute = client + method → provider
type ProviderRoute struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ClientID  uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_client_method"`
	Method    string    `gorm:"size:50;uniqueIndex:idx_client_method"` // "va_bca", "qris"
	Provider  string    `gorm:"size:50"`                               // "bca", "espay", "midtrans"
	IsActive  bool      `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type PaymentStatus string

const (
	StatusPending   PaymentStatus = "PENDING"
	StatusPaid      PaymentStatus = "PAID"
	StatusExpired   PaymentStatus = "EXPIRED"
	StatusFailed    PaymentStatus = "FAILED"
	StatusCancelled PaymentStatus = "CANCELLED"
)

type Payment struct {
	ID             uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ClientID       uuid.UUID     `gorm:"type:uuid;index;uniqueIndex:idx_client_ref"`
	ReferenceID    string        `gorm:"size:100;uniqueIndex:idx_client_ref"` // order id dari app (idempotency)
	Method         string        `gorm:"size:50"`
	Provider       string        `gorm:"size:50;index"`
	ProviderRef    string        `gorm:"size:150;index"`  // id transaksi di provider
	Amount         int64                                   // dalam rupiah, tanpa desimal
	Currency       string        `gorm:"size:3;default:IDR"`
	Status         PaymentStatus `gorm:"size:20;index"`
	VANumber       *string       `gorm:"size:50;index"`
	QRString       *string       `gorm:"type:text"`
	CustomerName   string        `gorm:"size:255"`
	CustomerEmail  string        `gorm:"size:255"`
	Description    string        `gorm:"size:500"`
	Metadata       datatypes.JSON
	ExpiredAt      *time.Time
	PaidAt         *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// CallbackLog = raw payload dari provider (audit)
type CallbackLog struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Provider   string    `gorm:"size:50;index"`
	PaymentID  *uuid.UUID `gorm:"type:uuid;index"`
	Headers    datatypes.JSON
	Payload    datatypes.JSON
	IsValid    bool
	Error      string `gorm:"type:text"`
	CreatedAt  time.Time
}

// WebhookLog = pengiriman webhook ke app + retry
type WebhookLog struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PaymentID    uuid.UUID `gorm:"type:uuid;index"`
	ClientID     uuid.UUID `gorm:"type:uuid;index"`
	Event        string    `gorm:"size:50"` // "payment.paid", "payment.expired"
	Payload      datatypes.JSON
	Attempt      int
	ResponseCode int
	ResponseBody string `gorm:"type:text"`
	Status       string `gorm:"size:20;index"` // PENDING, SUCCESS, FAILED
	NextRetryAt  *time.Time `gorm:"index"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
```

---

## 6. Abstraksi Provider (`libs/providers/provider.go`)

```go
type Provider interface {
	Name() string
	CreateVA(ctx context.Context, req CreateVARequest) (*PaymentResult, error)
	CreateQRIS(ctx context.Context, req CreateQRISRequest) (*PaymentResult, error)
	CheckStatus(ctx context.Context, providerRef string) (*StatusResult, error)
	Cancel(ctx context.Context, providerRef string) error
	ParseCallback(r *http.Request, body []byte) (*CallbackEvent, error) // verifikasi signature + normalisasi
	CallbackResponse(ev *CallbackEvent, err error) (int, any)           // format balasan sesuai spec provider
}

var ErrNotSupported = errors.New("method not supported by provider")

type CallbackEvent struct {
	Provider    string
	ProviderRef string
	VANumber    string
	Amount      int64
	Status      models.PaymentStatus
	PaidAt      *time.Time
	Raw         []byte
}

// Registry
func Get(name string) (Provider, error)
func Register(p Provider)
```

Aturan:
- Status provider **wajib dipetakan** ke `PaymentStatus` internal. Status mentah provider tidak boleh bocor ke app.
- Method yang tidak didukung provider → `ErrNotSupported`.
- Kredensial provider dari env/config, **tidak di-hardcode**.

---

## 7. Endpoint

### 7.1 Untuk App Client (`ClientAuthMiddleware`)
| Method | Path | Fungsi |
|---|---|---|
| POST | `/payments` | Buat transaksi (VA/QRIS). Idempotent berdasarkan `reference_id` |
| GET | `/payments/:id` | Detail + status |
| GET | `/payments` | List transaksi milik client |
| POST | `/payments/:id/cancel` | Batalkan (Delete-VA / cancel QRIS) |
| POST | `/payments/:id/check-status` | Paksa cek status ke provider |

`CreatePaymentDTO`:
```go
type CreatePaymentDTO struct {
	ReferenceID   string         `json:"reference_id" binding:"required,max=100"`
	Method        string         `json:"method" binding:"required,oneof=va_bca qris"`
	Amount        int64          `json:"amount" binding:"required,min=1"`
	CustomerName  string         `json:"customer_name" binding:"required,max=255"`
	CustomerEmail string         `json:"customer_email" binding:"omitempty,email"`
	Description   string         `json:"description" binding:"omitempty,max=500"`
	ExpiredMinutes int           `json:"expired_minutes" binding:"omitempty,min=5,max=10080"`
	Metadata      map[string]any `json:"metadata" binding:"omitempty"`
}
```

### 7.2 Callback dari Provider (public, tanpa JWT)
| Method | Path | Fungsi |
|---|---|---|
| POST | `/callbacks/bca/va/inquiry` | Inquiry VA dari BCA (jika diwajibkan) |
| POST | `/callbacks/bca/va/payment` | Payment Flag / H2H dari BCA |
| POST | `/callbacks/bca/qris/notify` | Notifikasi pembayaran QRIS BCA |
| POST | `/callbacks/espay/...` | (fase berikutnya) |
| POST | `/callbacks/midtrans/notify` | (fase berikutnya) |

### 7.3 Admin (`JWTMiddleware` + `AccessMiddleware`)
CRUD standar untuk `/clients`, `/provider-routes`, `/webhook-logs` (read-only + retry manual).

---

## 8. Alur Proses

### 8.1 Create Payment
1. `ClientAuthMiddleware` validasi API key + HMAC → set `client_id` di context.
2. Cek `reference_id` sudah ada untuk client ini → kembalikan payment lama (idempotent).
3. Cari `ProviderRoute` (client + method) → ambil provider dari registry.
4. Panggil `CreateVA` / `CreateQRIS`.
5. Simpan `Payment` status `PENDING`, return `va_number` / `qr_string` + `expired_at`.

### 8.2 Callback (contoh BCA Payment Flag)
1. Baca raw body, simpan ke `CallbackLog`.
2. `provider.ParseCallback` → verifikasi `X-SIGNATURE` + `X-TIMESTAMP`. Gagal → simpan `IsValid=false`, balas error sesuai spec.
3. Cari `Payment` via `ProviderRef` / `VANumber`.
4. **Idempotent:** jika sudah `PAID`, langsung balas sukses tanpa proses ulang.
5. Validasi `Amount` cocok.
6. Dalam **satu DB transaction**: update `Payment` → `PAID`, set `PaidAt`, insert `WebhookLog` status `PENDING` (outbox pattern).
7. Balas ke provider **secepatnya** dengan `CallbackResponse`.

### 8.3 Webhook ke App (worker)
- Worker ambil `WebhookLog` `PENDING` dengan `NextRetryAt <= now`.
- POST ke `client.CallbackURL` dengan header:
  - `X-JaxPay-Event: payment.paid`
  - `X-JaxPay-Timestamp: <unix>`
  - `X-JaxPay-Signature: HMAC-SHA256(webhook_secret, timestamp + "." + body)`
- 2xx → `SUCCESS`. Selain itu → retry dengan backoff (1m, 5m, 15m, 1h, 6h), maksimal 8 attempt → `FAILED`.

Payload webhook:
```json
{
  "event": "payment.paid",
  "data": {
    "id": "uuid",
    "reference_id": "ORDER-123",
    "method": "va_bca",
    "amount": 150000,
    "status": "PAID",
    "paid_at": "2026-11-03T14:00:00+07:00",
    "metadata": {}
  }
}
```

### 8.4 Worker Lain
- **Expire job:** `PENDING` dengan `ExpiredAt < now` → `EXPIRED` + webhook `payment.expired`.
- **Reconcile job:** `PENDING` > N menit → `CheckStatus` ke provider (jaring pengaman jika callback hilang).

---

## 9. Keamanan

- Semua request client wajib HMAC signature + timestamp (tolak jika selisih > 5 menit).
- Callback provider: verifikasi signature, opsional whitelist IP provider.
- Secret disimpan hash/terenkripsi, tidak pernah dikembalikan di response API.
- Jangan log kredensial atau token provider.
- Amount selalu `int64` rupiah — **tidak pakai float**.
- Rate limit per client di `app/gateway/rate_limiter.go`.

---

## 10. Environment Variables

```
APP_PORT=8080
DATABASE_URL=postgres://...
JWT_SECRET=...
REDIS_URL=...

BCA_BASE_URL=https://sandbox.bca.co.id
BCA_CLIENT_ID=...
BCA_CLIENT_SECRET=...
BCA_PRIVATE_KEY_PATH=./keys/bca_private.pem
BCA_PUBLIC_KEY_PATH=./keys/bca_public.pem
BCA_PARTNER_ID=...
BCA_CHANNEL_ID=...
BCA_COMPANY_CODE=...
```

---

## 11. Prompt untuk AI Assistant

> Kamu membangun **JaxPay**, payment gateway service berbasis Go (Gin + GORM + PostgreSQL) yang dipakai bersama oleh JaxLab, JAHRIS, dan Qrupi. Ikuti **seluruh** aturan di dokumen `JAXPAY_ARCHITECTURE.md`:
> - Struktur folder `app/`, `libs/`, `src/` persis seperti bagian 3.
> - Setiap module di `src/` hanya punya `controller.go`, `dto.go`, `module.go`, `service.go` dengan konvensi bagian 4 (variabel `x`, `d`, `v`, `e`; method service unexported; `helpers.RespondSuccess/RespondError`; anotasi Swagger lengkap).
> - Integrasi provider wajib lewat interface `Provider` di `libs/providers`, tidak boleh memanggil API bank langsung dari `src/`.
> - Callback harus idempotent, tersimpan di `CallbackLog`, dan update status + insert `WebhookLog` dalam satu transaksi.
> - Amount `int64`, status hanya dari `PaymentStatus` internal.
>
> Tugas saat ini: **[isi tugas, misalnya "buat module src/payments lengkap" atau "buat adapter libs/providers/bca untuk VA Payment Flag"]**.
> Tulis kode lengkap per file, jangan dipotong.
