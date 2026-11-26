package espay

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"jaxpay-go-backend/libs/config"
)

func TestSignCallbackVerifiesWhenSelfSignedAllowed(t *testing.T) {
	merchant, _ := rsa.GenerateKey(rand.Reader, 2048)
	espayKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	c := &Client{cfg: config.EspayConfig{AllowSelfSigned: true}, privKey: merchant, espayPubKey: &espayKey.PublicKey}
	path := "/api/webhooks/espay/v1.0/transfer-va/payment"
	body := []byte(`{ "virtualAccountNo": "JXP1" }`)

	ts, sig, _, e := c.SignCallback("POST", path, body)
	if e != nil {
		t.Fatal(e)
	}
	if !freshTimestamp(ts) || !c.verify("POST", path, []byte(`{"virtualAccountNo":"JXP1"}`), ts, sig) {
		t.Fatal("self-signed callback should verify")
	}

	c.cfg.AllowSelfSigned = false
	if c.verify("POST", path, body, ts, sig) {
		t.Fatal("self-signed callback must be rejected in production")
	}
	if _, _, _, e := c.SignCallback("POST", path, body); e == nil {
		t.Fatal("SignCallback must be disabled in production")
	}
}
