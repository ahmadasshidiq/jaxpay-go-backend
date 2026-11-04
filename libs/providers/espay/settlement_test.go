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
	c := &Client{cfg: config.EspayConfig{Password: "SECRET"}}
	form := url.Values{
		"rq_uuid":     {"uuid-1"},
		"rq_datetime": {"2024-03-14 07:49:28"},
		"password":    {"SECRET"},
		"data":        {`{"tx_id":"T1","payment_id":"P1","settlement_amount":"150000.00","settlement_date":"2024-03-14","settlement_remark":"ok"}`},
		"signature":   {"x"},
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

	form.Set("password", "WRONG")
	if _, e := send(form); e != providers.ErrInvalidSignature {
		t.Fatal("wrong password accepted")
	}
}
