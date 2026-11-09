package espay

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http/httptest"
	"strings"
	"testing"

	"jaxpay-go-backend/libs/models"
	"jaxpay-go-backend/libs/providers"
)

func TestVAPaymentCallback(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	c := &Client{privKey: k, espayPubKey: &k.PublicKey}
	path := "/api/v1/callbacks/espay/payment"
	send := func(status, sigOverride string) (*providers.CallbackEvent, error) {
		body := []byte(`{"partnerServiceId":" Espay","customerNo":"SGW","virtualAccountNo":"JXP1","trxId":"T1","paymentRequestId":"pay-1",` +
			`"paidAmount":{"value":"155000.00","currency":"IDR"},"totalAmount":{"value":"150000.00","currency":"IDR"},` +
			`"trxDateTime":"2024-03-14T07:49:28+07:00","additionalInfo":{"transactionStatus":"` + status + `"}}`)
		ts := timestamp()
		sig, _ := c.sign("POST", path, body, ts)
		if sigOverride != "" {
			sig = sigOverride
		}
		r := httptest.NewRequest("POST", path, strings.NewReader(string(body)))
		r.Header.Set("X-TIMESTAMP", ts)
		r.Header.Set("X-SIGNATURE", sig)
		return c.ParseCallback(r, body)
	}

	ev, e := send("S", "")
	if e != nil || ev.Type != providers.CallbackPayment || ev.ProviderRef != "JXP1" || ev.Amount != 150000 || ev.Status != models.StatusPaid || ev.PaidAt == nil {
		t.Fatalf("parse: %v %+v", e, ev)
	}
	ev.Bill = &providers.Bill{CustomerName: "Budi"}
	code, v := c.CallbackResponse(ev, nil)
	res := v.(map[string]any)
	va := res["virtualAccountData"].(map[string]any)
	info := res["additionalInfo"].(map[string]string)
	if code != 200 || res["responseCode"] != "2002500" || va["virtualAccountName"] != "Budi" || va["paymentRequestId"] != "pay-1" || len(info["reconcileId"]) != 32 || info["reconcileDatetime"] == "" {
		t.Fatalf("response %d %+v", code, res)
	}

	for s, want := range map[string]models.PaymentStatus{"F": models.StatusFailed, "IP": models.StatusPending, "SP": models.StatusPending, "HP": models.StatusPending} {
		if ev, e := send(s, ""); e != nil || ev.Status != want || ev.PaidAt != nil {
			t.Fatalf("%s: %v %+v", s, e, ev)
		}
	}
	if _, e := send("", ""); e != providers.ErrInvalidRequest {
		t.Fatalf("missing status: %v", e)
	}
	ev, e = send("S", "AAAA")
	if e != providers.ErrInvalidSignature {
		t.Fatalf("bad signature: %v", e)
	}
	if _, v := c.CallbackResponse(ev, e); v.(map[string]any)["responseCode"] != "4012500" {
		t.Fatal("signature code")
	}
}
