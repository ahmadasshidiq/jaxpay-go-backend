package espay

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"jaxpay-go-backend/libs/config"
)

const name = "espay"

var wib = time.FixedZone("WIB", 7*3600)

const tsLayout = "2006-01-02T15:04:05-07:00"

// Client adalah adapter Espay SNAP (QRIS MPM) dengan asymmetric signature SHA256withRSA.
type Client struct {
	cfg         config.EspayConfig
	http        *http.Client
	privKey     *rsa.PrivateKey // private key merchant: menandatangani request ke Espay
	espayPubKey *rsa.PublicKey  // public key Espay: memvalidasi callback dari Espay
}

func New(cfg config.EspayConfig) (*Client, error) {
	priv, e := loadPrivateKey(cfg.PrivateKeyPath)
	if e != nil {
		return nil, fmt.Errorf("espay private key: %w", e)
	}
	pub, e := loadPublicKey(cfg.PublicKeyPath)
	if e != nil {
		return nil, fmt.Errorf("espay public key: %w", e)
	}
	return &Client{cfg: cfg, http: &http.Client{Timeout: 30 * time.Second}, privKey: priv, espayPubKey: pub}, nil
}

func (c *Client) Name() string { return name }

// stringToSign = HTTPMethod:RelativeUrl:Lowercase(HexSHA256(MinifyJson(body))):Timestamp
func stringToSign(method, path string, body []byte, ts string) string {
	var min bytes.Buffer
	if len(body) > 0 && json.Compact(&min, body) != nil {
		min.Reset()
		min.Write(body)
	}
	h := sha256.Sum256(min.Bytes())
	return method + ":" + path + ":" + hex.EncodeToString(h[:]) + ":" + ts
}

// sign = base64(SHA256withRSA(privateKey, stringToSign))
func (c *Client) sign(method, path string, body []byte, ts string) (string, error) {
	sts := stringToSign(method, path, body, ts)
	digest := sha256.Sum256([]byte(sts))
	sig, e := rsa.SignPKCS1v15(rand.Reader, c.privKey, crypto.SHA256, digest[:])
	if e != nil {
		return "", e
	}
	out := base64.StdEncoding.EncodeToString(sig)
	if os.Getenv("ESPAY_DEBUG") != "" {
		var min bytes.Buffer
		_ = json.Compact(&min, body)
		bodyHash := sha256.Sum256(min.Bytes())
		pub, _ := x509.MarshalPKIXPublicKey(&c.privKey.PublicKey)
		fp := sha256.Sum256(pub)
		log.Printf("[espay] sign\n1. minifiedBody: %s\n2. lowercase(hex(sha256(minifiedBody))): %x\n3. stringToSign: %s\n4. sha256(stringToSign): %x\n5. signature (base64 SHA256withRSA): %s\nalgorithm: SHA256withRSA PKCS#1 v1.5, key %d-bit\nmerchant public key sha256 fingerprint: %x",
			min.Bytes(), bodyHash, sts, digest, out, c.privKey.N.BitLen(), fp)
	}
	return out, nil
}

// verify memvalidasi X-SIGNATURE dari Espay dengan public key Espay.
func (c *Client) verify(method, path string, body []byte, ts, signature string) bool {
	sig, e := base64.StdEncoding.DecodeString(signature)
	if e != nil {
		return false
	}
	digest := sha256.Sum256([]byte(stringToSign(method, path, body, ts)))
	return rsa.VerifyPKCS1v15(c.espayPubKey, crypto.SHA256, digest[:], sig) == nil
}

// do mengirim request SNAP ke Espay. path = relative URL, mis. /api/v1.0/qr/qr-mpm-generate.
func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	// body dikirim apa adanya (tidak di-minify); minify hanya dilakukan di stringToSign
	body, e := json.MarshalIndent(in, "", "  ")
	if e != nil {
		return e
	}
	ts := timestamp()
	sig, e := c.sign(method, path, body, ts)
	if e != nil {
		return e
	}
	req, e := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.cfg.BaseURL, "/")+path, bytes.NewReader(body))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TIMESTAMP", ts)
	req.Header.Set("X-SIGNATURE", sig)
	req.Header.Set("X-PARTNER-ID", c.cfg.PartnerID)
	req.Header.Set("X-EXTERNAL-ID", externalID())
	req.Header.Set("CHANNEL-ID", c.cfg.ChannelID)

	debug := os.Getenv("ESPAY_DEBUG") != ""
	if debug {
		log.Printf("[espay] %s %s\nheaders: %v\nbody: %s\nstringToSign: %s", method, req.URL, req.Header, body, stringToSign(method, path, body, ts))
	}

	res, e := c.http.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if e != nil {
		return e
	}
	if debug {
		log.Printf("[espay] response http %d: %s", res.StatusCode, raw)
	}
	if e = json.Unmarshal(raw, out); e != nil {
		return fmt.Errorf("espay %s: http %d: %w", path, res.StatusCode, e)
	}
	return nil
}

type snapResponse struct {
	ResponseCode    string `json:"responseCode"`
	ResponseMessage string `json:"responseMessage"`
}

func (r snapResponse) ok() bool { return strings.HasPrefix(r.ResponseCode, "200") }

func (r snapResponse) err(op string) error {
	return fmt.Errorf("espay %s: %s %s", op, r.ResponseCode, r.ResponseMessage)
}

func timestamp() string { return time.Now().In(wib).Format(tsLayout) }

func externalID() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(1e15))
	return fmt.Sprintf("%d%015d", time.Now().Unix()%100000, n.Int64())
}

type amount struct {
	Value    string `json:"value"`
	Currency string `json:"currency"`
}

func formatAmount(v int64) amount { return amount{Value: fmt.Sprintf("%d.00", v), Currency: "IDR"} }

// parseAmount mengubah "150000.00" → 150000 tanpa float; desimal non-nol ditolak.
func parseAmount(s string) (int64, error) {
	whole, frac, _ := strings.Cut(strings.TrimSpace(s), ".")
	if strings.Trim(frac, "0") != "" {
		return 0, fmt.Errorf("non-integer amount %q", s)
	}
	return strconv.ParseInt(whole, 10, 64)
}

// parseTS menerima +07:00 maupun +0700 (callback Espay memakai keduanya).
func parseTS(s string) (time.Time, error) {
	if t, e := time.Parse(tsLayout, s); e == nil {
		return t, nil
	}
	return time.Parse("2006-01-02T15:04:05-0700", s)
}

func parseTime(s string) *time.Time {
	if t, e := parseTS(s); e == nil {
		return &t
	}
	now := time.Now()
	return &now
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	blk, e := readPEM(path)
	if e != nil {
		return nil, e
	}
	if k, e := x509.ParsePKCS1PrivateKey(blk.Bytes); e == nil {
		return k, nil
	}
	k, e := x509.ParsePKCS8PrivateKey(blk.Bytes)
	if e != nil {
		return nil, e
	}
	rk, ok := k.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("not an RSA key")
	}
	return rk, nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	blk, e := readPEM(path)
	if e != nil {
		return nil, e
	}
	if k, e := x509.ParsePKIXPublicKey(blk.Bytes); e == nil {
		if rk, ok := k.(*rsa.PublicKey); ok {
			return rk, nil
		}
		return nil, errors.New("not an RSA key")
	}
	if cert, e := x509.ParseCertificate(blk.Bytes); e == nil {
		if rk, ok := cert.PublicKey.(*rsa.PublicKey); ok {
			return rk, nil
		}
	}
	return x509.ParsePKCS1PublicKey(blk.Bytes)
}

func readPEM(path string) (*pem.Block, error) {
	if path == "" {
		return nil, errors.New("path not configured")
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	blk, _ := pem.Decode(b)
	if blk == nil {
		return nil, errors.New("invalid PEM")
	}
	return blk, nil
}
