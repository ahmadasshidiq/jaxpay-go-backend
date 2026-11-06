package espay

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"jaxpay-go-backend/libs/config"
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
		Amount: 150000, CustomerName: "Yessi Shop!", BankCode: "014", ExpiredAt: time.Now().Add(time.Hour),
	})
	if e != nil {
		t.Fatal(e)
	}
	if *res.VANumber != "8920800000012345" || res.ProviderRef != got.Get("order_id") {
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

// Contoh Format Signature Universal (Send Invoice) dari dokumentasi Espay.
func TestHashSignatureMatchesEspayDocs(t *testing.T) {
	got := hashSignature("cc256d3a2d7687e6f4e1f4217c534bc6b18f66e3552aa9d312f5f4808130504", "SENDINVOICE",
		"rfbd39734-ed32-490d-98c4-e91bcd91037a", "2024-01-01 14:39:11", "ORDER001", "100000", "IDR", "SGWDIGALLERY")
	if want := "b474188c95439412262f5808473caa8c12676acf4381842ff43b1b4a22493808"; got != want {
		t.Fatalf("signature\n got %s\nwant %s", got, want)
	}
}
