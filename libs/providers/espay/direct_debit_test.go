package espay

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jaxpay-go-backend/libs/config"
	"jaxpay-go-backend/libs/providers"
)

func TestCreateDirectDebit(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	resp := `{"responseCode":"2005400","responseMessage":"Success","webRedirectUrl":"https://pay.test/x"}`
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		c := &Client{espayPubKey: &k.PublicKey}
		if r.URL.Path != pathDirectDebit || !c.verify("POST", pathDirectDebit, raw, r.Header.Get("X-TIMESTAMP"), r.Header.Get("X-SIGNATURE")) {
			t.Errorf("request %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(resp))
	}))
	defer srv.Close()
	c := &Client{cfg: config.EspayConfig{BaseURL: "http://invalid.test", MerchantPGURL: srv.URL, MerchantID: "SGW"}, http: srv.Client(), privKey: k}
	req := providers.CreateDirectDebitRequest{
		ProviderRef: "JXP1", Amount: 150000, BankCode: "014", ProductCode: "GOPAYLINK", ReturnURL: "https://m.test/r", ExpiredAt: time.Now().Add(time.Hour),
	}
	res, e := c.CreateDirectDebit(context.Background(), req)
	if e != nil || res.ProviderRef != "JXP1" || *res.RedirectURL != "https://pay.test/x" {
		t.Fatalf("%v %+v", e, res)
	}
	opt := body["payOptionDetails"].(map[string]any)
	url := body["urlParam"].(map[string]any)
	info := body["additionalInfo"].(map[string]any)
	if body["partnerReferenceNo"] != "JXP1" || opt["payMethod"] != "014" || opt["payOption"] != "GOPAYLINK" || url["isDeeplink"] != "N" || url["type"] != "PAY_RETURN" || info["payType"] != "REDIRECT" {
		t.Fatalf("body %v", body)
	}
	resp = `{"responseCode":"4005401","responseMessage":"Invalid"}`
	if _, e := c.CreateDirectDebit(context.Background(), req); e == nil {
		t.Fatal("error response accepted")
	}
	if e := c.Cancel(context.Background(), "direct_debit", "JXP1"); !errors.Is(e, providers.ErrNotSupported) {
		t.Fatalf("cancel: %v", e)
	}
}
