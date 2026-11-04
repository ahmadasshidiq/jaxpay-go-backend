package espay

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
)

// Contoh dari dokumentasi Espay.
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

// Contoh validasi signature callback (VA inquiry) dari dokumentasi Espay.
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
