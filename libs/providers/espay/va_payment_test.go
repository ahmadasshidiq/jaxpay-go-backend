package espay

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
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
	body := v.(vaPaymentBody)
	if len(body.AdditionalInfo.ReconcileID) != 32 || body.AdditionalInfo.ReconcileDatetime == "" {
		t.Fatalf("reconcile: %+v", body.AdditionalInfo)
	}
	body.AdditionalInfo = reconcileInfo{}
	got, _ := json.Marshal(body)
	want := `{"responseCode":"2002500","responseMessage":"Success","virtualAccountData":{"partnerServiceId":" Espay","customerNo":"SGW","virtualAccountNo":"JXP1","virtualAccountName":"Budi","paymentRequestId":"pay-1","totalAmount":{"value":"150000.00","currency":"IDR"},"billDetails":[{"billDescription":{"english":"Pembayaran","indonesia":"Pembayaran"}}]},"additionalInfo":{"reconcileId":"","reconcileDatetime":""}}`
	if code != 200 || string(got) != want {
		t.Fatalf("response %d\n got %s\nwant %s", code, got, want)
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
	code, v = c.CallbackResponse(ev, e)
	if got, _ := json.Marshal(v); code != 401 || string(got) != `{"responseCode":"4012500","responseMessage":"Unauthorized Signature"}` {
		t.Fatalf("signature error: %d %s", code, got)
	}
}
