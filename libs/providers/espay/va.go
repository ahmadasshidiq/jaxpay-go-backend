package espay

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"jaxpay-go-backend/libs/providers"
)

const pathSendInvoice = "/rest/merchantpg/sendinvoice"

const dtLayout = "2006-01-02 15:04:05"

// CreateVA membuat Virtual Account dinamis (non-fixed) via Direct API sendinvoice.
// order_id dipakai sebagai provider_ref sehingga callback inquiry/payment bisa menemukan payment-nya.
func (c *Client) CreateVA(ctx context.Context, req providers.CreateVARequest) (*providers.PaymentResult, error) {
	if c.cfg.SignatureKey == "" {
		return nil, errors.New("espay sendinvoice: signature key not configured")
	}
	if req.BankCode == "" {
		return nil, fmt.Errorf("%w: bank_code required", providers.ErrInvalidRequest)
	}
	orderID := req.ProviderRef
	if orderID == "" {
		orderID = providers.NewProviderRef()
	}
	rqUUID := newUUID()
	rqDatetime := time.Now().In(wib).Format(dtLayout)
	amt := fmt.Sprintf("%d.00", req.Amount)

	f := url.Values{}
	f.Set("rq_uuid", rqUUID)
	f.Set("rq_datetime", rqDatetime)
	f.Set("order_id", orderID)
	f.Set("amount", amt)
	f.Set("ccy", "IDR")
	f.Set("comm_code", c.cfg.MerchantID)
	f.Set("remark1", trunc(req.CustomerPhone, 17))
	f.Set("remark2", trunc(alnumSpace(req.CustomerName), 30))
	f.Set("remark3", trunc(req.CustomerEmail, 50))
	f.Set("update", "N")
	f.Set("bank_code", req.BankCode)
	if !req.ExpiredAt.IsZero() {
		f.Set("va_expired", strconv.Itoa(int(math.Ceil(time.Until(req.ExpiredAt).Minutes()))))
	}
	f.Set("signature", hashSignature(c.cfg.SignatureKey, "SENDINVOICE", rqUUID, rqDatetime, orderID, amt, "IDR", c.cfg.MerchantID))

	var out struct {
		ErrorCode    string `json:"error_code"`
		ErrorMessage string `json:"error_message"`
		VANumber     string `json:"va_number"`
		Expired      string `json:"expired"`
	}
	if e := c.postForm(ctx, pathSendInvoice, f, &out); e != nil {
		return nil, e
	}
	if out.ErrorCode != "0000" || out.VANumber == "" {
		return nil, fmt.Errorf("espay sendinvoice: %s %s", out.ErrorCode, out.ErrorMessage)
	}
	res := &providers.PaymentResult{ProviderRef: orderID, VANumber: &out.VANumber}
	if t, e := time.ParseInLocation(dtLayout, out.Expired, wib); e == nil {
		res.ExpiredAt = &t
	} else if !req.ExpiredAt.IsZero() {
		res.ExpiredAt = &req.ExpiredAt
	}
	return res, nil
}

// hashSignature = Hash-Based Signature Espay:
// lowercase(hex(SHA256(UPPERCASE("##" + signatureKey + "##" + field1 + "##" + ... + "##" + mode + "##"))))
func hashSignature(key, mode string, fields ...string) string {
	s := "##" + key + "##" + strings.Join(fields, "##") + "##" + mode + "##"
	h := sha256.Sum256([]byte(strings.ToUpper(s)))
	return hex.EncodeToString(h[:])
}

// postForm mengirim request Direct API (non-SNAP) application/x-www-form-urlencoded.
func (c *Client) postForm(ctx context.Context, path string, f url.Values, out any) error {
	base := c.cfg.MerchantPGURL
	if base == "" {
		base = c.cfg.BaseURL
	}
	body := f.Encode()
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(base, "/")+path, strings.NewReader(body))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "*/*")

	debug := os.Getenv("ESPAY_DEBUG") != ""
	if debug {
		log.Printf("[espay] POST %s\nbody: %s", req.URL, body)
	}
	res, e := c.http.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if e != nil {
		return e
	}
	if debug {
		log.Printf("[espay] response http %d: %s", res.StatusCode, raw)
	}
	if e = json.Unmarshal(raw, out); e != nil {
		return fmt.Errorf("espay %s: http %d: %w", path, res.StatusCode, e)
	}
	return nil
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	h := hex.EncodeToString(b)
	return h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}

// alnumSpace menyisakan huruf, angka, dan spasi (format remark2).
func alnumSpace(s string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' {
			return r
		}
		return -1
	}, s)
}
