package espay

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"jaxpay-go-backend/libs/config"
	"jaxpay-go-backend/libs/models"
	"jaxpay-go-backend/libs/providers"
)

func TestCreateVASendInvoice(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != pathSendInvoice || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("unexpected request %s %s", r.URL.Path, r.Header.Get("Content-Type"))
		}
		_ = r.ParseForm()
		got = r.PostForm
		_, _ = w.Write([]byte(`{"rq_uuid":"x","error_code":"0000","error_message":"","va_number":"8920800000012345","expired":"2024-03-14 08:49:28","amount":"150000.00","fee":"0.00","total_amount":"150000.00"}`))
	}))
	defer srv.Close()

	c := &Client{cfg: config.EspayConfig{BaseURL: srv.URL, MerchantID: "SGWYESSISHOP", SignatureKey: "secret"}, http: srv.Client()}
	res, e := c.CreateVA(context.Background(), providers.CreateVARequest{
		ProviderRef: "JXPABC123", Amount: 150000, CustomerName: "Yessi Shop!", BankCode: "014", ExpiredAt: time.Now().Add(time.Hour),
	})
	if e != nil {
		t.Fatal(e)
	}
	if *res.VANumber != "8920800000012345" || res.ProviderRef != "JXPABC123" || got.Get("order_id") != "JXPABC123" {
		t.Fatalf("result %+v", res)
	}
	if got.Get("amount") != "150000.00" || got.Get("remark2") != "Yessi Shop" || got.Get("va_expired") != "60" || got.Get("update") != "N" {
		t.Fatalf("form %v", got)
	}
	want := hashSignature("secret", "SENDINVOICE", got.Get("rq_uuid"), got.Get("rq_datetime"), got.Get("order_id"), "150000.00", "IDR", "SGWYESSISHOP")
	if got.Get("signature") != want {
		t.Fatalf("signature %s want %s", got.Get("signature"), want)
	}
}

func TestHashSignatureMatchesEspayDocs(t *testing.T) {
	got := hashSignature("cc256d3a2d7687e6f4e1f4217c534bc6b18f66e3552aa9d312f5f4808130504", "SENDINVOICE",
		"rfbd39734-ed32-490d-98c4-e91bcd91037a", "2024-01-01 14:39:11", "ORDER001", "100000", "IDR", "SGWDIGALLERY")
	if want := "b474188c95439412262f5808473caa8c12676acf4381842ff43b1b4a22493808"; got != want {
		t.Fatalf("signature\n got %s\nwant %s", got, want)
	}
}

func TestCheckStatusVA(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	resp := ""
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		if r.URL.Path != pathVAStatus {
			t.Errorf("path %s", r.URL.Path)
		}
		c := &Client{espayPubKey: &k.PublicKey}
		if !c.verify("POST", pathVAStatus, raw, r.Header.Get("X-TIMESTAMP"), r.Header.Get("X-SIGNATURE")) {
			t.Error("invalid signature")
		}
		_, _ = w.Write([]byte(resp))
	}))
	defer srv.Close()
	c := &Client{cfg: config.EspayConfig{BaseURL: "http://invalid.test", MerchantPGURL: srv.URL, MerchantID: "SGW"}, http: srv.Client(), privKey: k}

	ok := func(flag string) string {
		return `{"responseCode":"2002600","responseMessage":"Success","virtualAccountData":{"paymentFlagStatus":"` + flag +
			`","paidAmount":{"value":"150000.00","currency":"IDR"},"trxDateTime":"2024-03-14T07:49:28+07:00"}}`
	}
	want := map[string]models.PaymentStatus{
		"S": models.StatusPaid, "F": models.StatusFailed, "EX": models.StatusExpired, "IP": models.StatusPending,
		"SP": models.StatusPending, "WC": models.StatusPending, "WS": models.StatusPending,
	}
	for flag, status := range want {
		resp = ok(flag)
		r, e := c.CheckStatus(context.Background(), "va", "JXP1")
		if e != nil || r.Status != status {
			t.Fatalf("%s: %v %+v", flag, e, r)
		}
		if flag == "S" && (r.Amount != 150000 || r.PaidAt == nil || r.PaidAt.Unix() != 1710377368) {
			t.Fatalf("paid result %+v", r)
		}
	}
	if body["partnerServiceId"] != " ESPAY" || body["customerNo"] != "SGW" || body["virtualAccountNo"] != "JXP1" || body["additionalInfo"] != nil {
		t.Fatalf("body %v", body)
	}

	resp = `{"responseCode":"4042601","responseMessage":"Not found"}`
	if _, e := c.CheckStatus(context.Background(), "va", "JXP1"); e == nil {
		t.Fatal("error response accepted")
	}
	resp = `{"responseCode":"2002600","responseMessage":"Success","virtualAccountData":{"paymentFlagStatus":"S","paidAmount":{"value":"","currency":"IDR"}}}`
	if _, e := c.CheckStatus(context.Background(), "va", "JXP1"); e == nil {
		t.Fatal("paid without amount accepted")
	}
}

func TestCancelVA(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	resp := `{"responseCode":"2003100","responseMessage":"Success"}`
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		if r.Method != http.MethodDelete || r.URL.Path != pathVADelete {
			t.Errorf("request %s %s", r.Method, r.URL.Path)
		}
		c := &Client{espayPubKey: &k.PublicKey}
		if !c.verify(http.MethodDelete, pathVADelete, raw, r.Header.Get("X-TIMESTAMP"), r.Header.Get("X-SIGNATURE")) {
			t.Error("invalid signature")
		}
		_, _ = w.Write([]byte(resp))
	}))
	defer srv.Close()
	c := &Client{cfg: config.EspayConfig{BaseURL: "http://invalid.test", MerchantPGURL: srv.URL, MerchantID: "SGW"}, http: srv.Client(), privKey: k}

	if e := c.Cancel(context.Background(), "va", "JXP1"); e != nil {
		t.Fatal(e)
	}
	if body["partnerServiceId"] != " ESPAY" || body["customerNo"] != "SGW" || body["virtualAccountNo"] != "JXP1" {
		t.Fatalf("body %v", body)
	}
	resp = `{"responseCode":"4043100","responseMessage":"Not found"}`
	if e := c.Cancel(context.Background(), "va", "JXP1"); e == nil {
		t.Fatal("error response accepted")
	}
}
