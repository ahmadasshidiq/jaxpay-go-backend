package espay

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"jaxpay-go-backend/libs/providers"
)

func TestInquiryCallback(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	c := &Client{privKey: k, espayPubKey: &k.PublicKey}
	path := "/api/v1/callbacks/espay/inquiry"
	body := []byte(`{"partnerServiceId":" Espay","customerNo":"SGWJAXERTEKNOINDO","virtualAccountNo":"JXP123","trxDateInit":"2024-06-17T21:45:46+0700","inquiryRequestId":"req-1"}`)
	ts := timestamp()
	sig, _ := c.sign("POST", path, body, ts)

	r := httptest.NewRequest("POST", path, strings.NewReader(string(body)))
	r.Header.Set("X-TIMESTAMP", ts)
	r.Header.Set("X-SIGNATURE", sig)
	ev, e := c.ParseCallback(r, body)
	if e != nil || ev.Type != providers.CallbackInquiry || ev.ProviderRef != "JXP123" {
		t.Fatalf("parse: %v %+v", e, ev)
	}

	ev.Bill = &providers.Bill{CustomerName: "Budi", Amount: 150000, CreatedAt: time.Now()}
	code, v := c.CallbackResponse(ev, nil)
	res := v.(map[string]any)
	va := res["virtualAccountData"].(map[string]any)
	if code != 200 || res["responseCode"] != "2002400" || va["inquiryRequestId"] != "req-1" || va["totalAmount"].(amount).Value != "150000.00" {
		t.Fatalf("response: %d %+v", code, res)
	}

	if _, v := c.CallbackResponse(ev, providers.ErrAlreadyPaid); v.(map[string]any)["responseCode"] != "4042414" {
		t.Fatal("paid bill code")
	}
	r.Header.Set("X-SIGNATURE", "AAAA")
	if _, e := c.ParseCallback(r, body); e != providers.ErrInvalidSignature {
		t.Fatal("bad signature accepted")
	}
}
