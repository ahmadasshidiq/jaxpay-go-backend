package espay

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"jaxpay-go-backend/libs/config"
	"jaxpay-go-backend/libs/providers"
)

func TestStringToSignMatchesEspayDocs(t *testing.T) {
	body := []byte(`{
"partnerReferenceNo": "DIGSNAP000033",
"merchantId": "SGWYESSISHOP",
"amount": {
"value": "10001.00",
"currency": "IDR"
},
"additionalInfo": {
"productCode": "QRIS"
},
"validityPeriod": "2024-03-20T07:44:11+07:00"
}`)
	want := "POST:/api/v1.0/qr/qr-mpm-generate:19e872e9ac1eeda0fa08c9f710cb7cfd7a94336ec17b94126452fb4fda049a29:2024-03-14T07:49:28+07:00"
	if got := stringToSign("POST", "/api/v1.0/qr/qr-mpm-generate", body, "2024-03-14T07:49:28+07:00"); got != want {
		t.Fatalf("stringToSign\n got %s\nwant %s", got, want)
	}
}

func TestCallbackStringToSignMatchesEspayDocs(t *testing.T) {
	body := []byte(`{
  "partnerServiceId": " Espay",
  "customerNo": "SGWDIGALLERY",
  "virtualAccountNo": "DIGORDER000002",
  "trxDateInit": "2024-06-17T21:45:46+0700",
  "inquiryRequestId": "710c1836-eddf-49d1-8234-f11e93750fdd"
}`)
	path, ts := "/api/webhooks/epsay/v1.0/transfer-va/inquiry.php", "2024-06-17T21:45:46+0700"
	want := "POST:" + path + ":33578ff224ac535c2be314623a3ba420f6b965f4570ec9bbb8af17ac8dbd6468:" + ts
	if got := stringToSign("POST", path, body, ts); got != want {
		t.Fatalf("stringToSign\n got %s\nwant %s", got, want)
	}
	if _, e := parseTS(ts); e != nil {
		t.Fatalf("timestamp +0700 ditolak: %v", e)
	}
}

func TestSignVerifyRoundTrip(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	c := &Client{privKey: k, espayPubKey: &k.PublicKey}
	body := []byte(`{"a": 1}`)
	sig, e := c.sign("POST", "/x", body, "2024-03-14T07:49:28+07:00")
	if e != nil {
		t.Fatal(e)
	}
	if !c.verify("POST", "/x", []byte(`{"a":1}`), "2024-03-14T07:49:28+07:00", sig) {
		t.Fatal("valid signature rejected (minify must normalise whitespace)")
	}
	if c.verify("POST", "/x", []byte(`{"a":2}`), "2024-03-14T07:49:28+07:00", sig) {
		t.Fatal("tampered body accepted")
	}
}

func TestCreateQRISUsesProviderRef(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		_, _ = w.Write([]byte(`{"responseCode":"2024700","responseMessage":"ok","qrContent":"000201","qrUrl":"https://qr.test/1","additionalInfo":{"referenceNo":"ESP1","amount":"10000"}}`))
	}))
	defer srv.Close()
	c := &Client{cfg: config.EspayConfig{BaseURL: srv.URL}, http: srv.Client(), privKey: k}
	res, e := c.CreateQRIS(context.Background(), providers.CreateQRISRequest{ProviderRef: "JXP1", Amount: 10000})
	if e != nil || res.ProviderRef != "JXP1" || res.QRURL == nil || *res.QRURL != "https://qr.test/1" || body["partnerReferenceNo"] != "JXP1" || string(res.ProviderData) != `{"referenceNo":"ESP1","amount":"10000"}` {
		t.Fatalf("%v %+v %v", e, res, body)
	}
}

func TestSnapResponseOK(t *testing.T) {
	for code, want := range map[string]bool{"2004700": true, "2024700": true, "4004700": false, "5004700": false, "": false} {
		if got := (snapResponse{ResponseCode: code}).ok(); got != want {
			t.Errorf("%q: got %v", code, got)
		}
	}
}
