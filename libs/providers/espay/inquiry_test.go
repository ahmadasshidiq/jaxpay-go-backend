package espay

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
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

	ev.Bill = &providers.Bill{CustomerName: "Budi", CustomerEmail: "b@x.id", CustomerPhone: "628123", Description: "Invoice 1", Amount: 150000,
		CreatedAt: time.Date(2026, 11, 26, 10, 0, 0, 0, wib),
		Metadata:  []byte(`{"token":"T1","other":"x","dataMerchant":{"feeCa":5000,"listBills":[{"billCode":1,"billName":"B1","billAmount":{"value":"70000.00","currency":"IDR"},"billSubCompany":"00001"}]}}`)}
	code, v := c.CallbackResponse(ev, nil)
	got, _ := json.Marshal(v)
	want := `{"responseCode":"2002400","responseMessage":"Success","virtualAccountData":{"partnerServiceId":" Espay","customerNo":"SGWJAXERTEKNOINDO","virtualAccountNo":"JXP123","virtualAccountName":"Budi","virtualAccountEmail":"b@x.id","virtualAccountPhone":"628123","inquiryRequestId":"req-1","totalAmount":{"value":"150000.00","currency":"IDR"},"billDetails":[{"billDescription":{"english":"Invoice 1","indonesia":"Invoice 1"}}],"additionalInfo":{"token":"T1","transactionDate":"2026-11-26T10:00:00+07:00","dataMerchant":{"kodeCa":"","kodeSubCa":"","noKontrak":"","namaPelanggan":"","angsuranKe":"","jmlBayarExcAdm":0.00,"denda":0.00,"feeCa":5000.00,"feeSwitcher":0.00,"totalAdmin":0.00,"jumlahBayar":0.00,"minimumAmount":0.00,"totalAngsuran":0,"customer":{"email":""},"jatuhTempo":"","listBills":[{"billCode":1,"billName":"B1","billAmount":{"value":"70000.00","currency":"IDR"},"billSubCompany":"00001"}]},"shippingAddress":{"firstName":"","lastName":"","address":"","city":"","postalCode":"","phoneNumber":"","countryCode":""},"items":[]}}}`
	if code != 200 || string(got) != want {
		t.Fatalf("response\n got %s\nwant %s", got, want)
	}

	if _, v := c.CallbackResponse(ev, providers.ErrAlreadyPaid); v.(snapErrorBody).ResponseCode != "4042414" {
		t.Fatal("paid bill code")
	}
	code, v = c.CallbackResponse(ev, providers.ErrInvalidSignature)
	if got, _ := json.Marshal(v); code != 401 || string(got) != `{"responseCode":"4012400","responseMessage":"Unauthorized Signature"}` {
		t.Fatalf("signature error: %d %s", code, got)
	}
	r.Header.Set("X-SIGNATURE", "AAAA")
	if _, e := c.ParseCallback(r, body); e != providers.ErrInvalidSignature {
		t.Fatal("bad signature accepted")
	}
}
