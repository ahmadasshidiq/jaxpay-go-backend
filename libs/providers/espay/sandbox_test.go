package espay

import (
	"context"
	"os"
	"testing"
	"time"

	"jaxpay-go-backend/libs/config"
	"jaxpay-go-backend/libs/providers"

	"github.com/joho/godotenv"
)

// Hit sandbox Espay sungguhan. Jalankan: ESPAY_SANDBOX=1 go test ./libs/providers/espay -run Sandbox -v
func TestSandboxCreateQRIS(t *testing.T) {
	if os.Getenv("ESPAY_SANDBOX") == "" {
		t.Skip("set ESPAY_SANDBOX=1")
	}
	_ = os.Chdir("../../..") // path key di .env relatif ke root project
	_ = godotenv.Load()
	c, e := New(config.Get().Espay)
	if e != nil {
		t.Fatal(e)
	}
	res, e := c.CreateQRIS(context.Background(), providers.CreateQRISRequest{
		Amount:    10000,
		ExpiredAt: time.Now().Add(30 * time.Minute),
	})
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("ref=%s qr=%s", res.ProviderRef, *res.QRString)
}
