package espay

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"jaxpay-go-backend/libs/config"
	"jaxpay-go-backend/libs/providers"
)

func TestSettlementCallback(t *testing.T) {
	c := &Client{cfg: config.EspayConfig{Password: "SECRET", MerchantID: "SGWYESSISHOP"}}
	form := url.Values{
		"rq_uuid":     {"uuid-1"},
		"rq_datetime": {"2024-03-14 07:49:28"},
		"password":    {"SECRET"},
		"data":        {`{"tx_id":"T1","payment_id":"P1","settlement_amount":"150000.00","settlement_date":"2024-03-14","settlement_remark":"ok"}`},
		"signature":   {settlementSignature("uuid-1", "2024-03-14 07:49:28", defaultSettlementSender, "SGWYESSISHOP")},
	}
	send := func(f url.Values) (*providers.CallbackEvent, error) {
		body := []byte(f.Encode())
		r := httptest.NewRequest("POST", "/api/v1/callbacks/espay/settlement", strings.NewReader(string(body)))
		return c.ParseCallback(r, body)
	}

	ev, e := send(form)
	if e != nil || ev.Type != providers.CallbackSettlement || ev.ProviderRef != "P1" || ev.Amount != 150000 {
		t.Fatalf("parse: %v %+v", e, ev)
	}
	if _, v := c.CallbackResponse(ev, nil); v.(map[string]string)["error_code"] != "0000" {
		t.Fatal("success code")
	}

	form.Set("signature", "deadbeef")
	if _, e := send(form); e != providers.ErrInvalidSignature {
		t.Fatal("wrong signature accepted")
	}

	form.Set("signature", settlementSignature("uuid-1", "2024-03-14 07:49:28", "OTHER", "RCV"))
	form.Set("sender_id", "OTHER")
	form.Set("receiver_id", "RCV")
	if _, e := send(form); e != nil {
		t.Fatalf("sender_id/receiver_id dari form ditolak: %v", e)
	}

	form.Set("password", "WRONG")
	if _, e := send(form); e != providers.ErrInvalidSignature {
		t.Fatal("wrong password accepted")
	}
}

// Contoh Format Signature Settlement Notification dari dokumentasi Espay.
func TestSettlementSignatureMatchesEspayDocs(t *testing.T) {
	got := settlementSignature("cc256d3a2d7687e6f4e1f4217c534bc6b18f66e3552aa9d312f5f4808130504", "2024-01-01 14:39:11", "GOWORLDPG", "SGWYESSISHOP")
	if want := "591e6edde42e0d63705ccca9d7ff077392aa7f03"; got != want {
		t.Fatalf("signature\n got %s\nwant %s", got, want)
	}
}
