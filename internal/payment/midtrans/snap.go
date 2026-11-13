// Package midtrans implements the Midtrans Snap HTTP contract without coupling
// billing rules or the rest of MWX-ISP to a provider SDK.
package midtrans

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var ErrNotConfigured = errors.New("Midtrans payment provider is not configured")

type Config struct {
	ServerKey string
	ClientKey string
	BaseURL   string
	Timeout   time.Duration
}

type Customer struct {
	Name  string
	Email string
	Phone string
}

type Checkout struct {
	Token       string
	RedirectURL string
}

type Notification struct {
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
	TransactionStatus string `json:"transaction_status"`
	TransactionID     string `json:"transaction_id"`
	PaymentType       string `json:"payment_type"`
	FraudStatus       string `json:"fraud_status"`
	Currency          string `json:"currency"`
	StatusMessage     string `json:"status_message"`
}

type Client struct {
	config Config
	http   *http.Client
}

// ConfigFromEnv returns a provider configuration only when Midtrans is
// explicitly selected. Secrets are read at runtime and are never serialized.
func ConfigFromEnv() (Config, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("TOUGHRADIUS_PAYMENT_PROVIDER")))
	if provider == "" {
		return Config{}, ErrNotConfigured
	}
	if provider != "midtrans" {
		return Config{}, fmt.Errorf("unsupported payment provider %q", provider)
	}
	serverKey := strings.TrimSpace(os.Getenv("TOUGHRADIUS_MIDTRANS_SERVER_KEY"))
	clientKey := strings.TrimSpace(os.Getenv("TOUGHRADIUS_MIDTRANS_CLIENT_KEY"))
	if serverKey == "" || clientKey == "" {
		return Config{}, ErrNotConfigured
	}
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("TOUGHRADIUS_MIDTRANS_ENVIRONMENT")))
	if mode == "" {
		mode = "sandbox"
	}
	baseURL := strings.TrimSpace(os.Getenv("TOUGHRADIUS_MIDTRANS_BASE_URL"))
	if baseURL == "" {
		switch mode {
		case "sandbox":
			baseURL = "https://app.sandbox.midtrans.com"
		case "production":
			baseURL = "https://app.midtrans.com"
		default:
			return Config{}, fmt.Errorf("Midtrans environment must be sandbox or production")
		}
	}
	timeout := 10
	if raw := strings.TrimSpace(os.Getenv("TOUGHRADIUS_PAYMENT_TIMEOUT_SECONDS")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 60 {
			return Config{}, fmt.Errorf("payment timeout must be between 1 and 60 seconds")
		}
		timeout = parsed
	}
	return Config{ServerKey: serverKey, ClientKey: clientKey, BaseURL: baseURL, Timeout: time.Duration(timeout) * time.Second}, nil
}

func New(cfg Config, httpClient *http.Client) (*Client, error) {
	cfg.ServerKey = strings.TrimSpace(cfg.ServerKey)
	cfg.ClientKey = strings.TrimSpace(cfg.ClientKey)
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if cfg.ServerKey == "" || cfg.ClientKey == "" {
		return nil, ErrNotConfigured
	}
	base, err := url.Parse(cfg.BaseURL)
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") {
		return nil, fmt.Errorf("Midtrans base URL must be an absolute HTTP(S) URL")
	}
	if base.Scheme != "https" && base.Hostname() != "127.0.0.1" && base.Hostname() != "localhost" {
		return nil, fmt.Errorf("Midtrans base URL must use HTTPS")
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.Timeout > 60*time.Second {
		return nil, fmt.Errorf("Midtrans request timeout cannot exceed 60 seconds")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: cfg.Timeout}
	}
	return &Client{config: cfg, http: httpClient}, nil
}

type createRequest struct {
	TransactionDetails struct {
		OrderID     string `json:"order_id"`
		GrossAmount int64  `json:"gross_amount"`
	} `json:"transaction_details"`
	CreditCard struct {
		Secure bool `json:"secure"`
	} `json:"credit_card"`
	CustomerDetails struct {
		FirstName string `json:"first_name,omitempty"`
		Email     string `json:"email,omitempty"`
		Phone     string `json:"phone,omitempty"`
	} `json:"customer_details,omitempty"`
}

type createResponse struct {
	Token       string `json:"token"`
	RedirectURL string `json:"redirect_url"`
}

// CreateCheckout asks Midtrans for a Snap token. The order identifier and
// amount must already have been derived from and persisted against an invoice.
func (c *Client) CreateCheckout(ctx context.Context, orderID string, amount int64, customer Customer) (Checkout, error) {
	if len(orderID) == 0 || len(orderID) > 50 || amount <= 0 {
		return Checkout{}, fmt.Errorf("invalid Midtrans order identifier or amount")
	}
	for _, r := range orderID {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("-_.~", r)) {
			return Checkout{}, fmt.Errorf("invalid Midtrans order identifier")
		}
	}
	var payload createRequest
	payload.TransactionDetails.OrderID = orderID
	payload.TransactionDetails.GrossAmount = amount
	payload.CreditCard.Secure = true
	payload.CustomerDetails.FirstName = strings.TrimSpace(customer.Name)
	payload.CustomerDetails.Email = strings.TrimSpace(customer.Email)
	payload.CustomerDetails.Phone = strings.TrimSpace(customer.Phone)
	body, err := json.Marshal(payload)
	if err != nil {
		return Checkout{}, fmt.Errorf("encode Midtrans checkout: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/snap/v1/transactions", bytes.NewReader(body))
	if err != nil {
		return Checkout{}, fmt.Errorf("build Midtrans request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(c.config.ServerKey+":")))
	res, err := c.http.Do(req)
	if err != nil {
		return Checkout{}, fmt.Errorf("Midtrans Snap request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
		return Checkout{}, fmt.Errorf("Midtrans Snap returned HTTP %d", res.StatusCode)
	}
	var decoded createResponse
	if err := json.NewDecoder(io.LimitReader(res.Body, 64*1024)).Decode(&decoded); err != nil {
		return Checkout{}, fmt.Errorf("decode Midtrans Snap response: %w", err)
	}
	if strings.TrimSpace(decoded.Token) == "" || !validRedirect(c.config.BaseURL, decoded.RedirectURL) {
		return Checkout{}, fmt.Errorf("Midtrans returned an invalid checkout response")
	}
	return Checkout{Token: decoded.Token, RedirectURL: decoded.RedirectURL}, nil
}

func validRedirect(baseURL, redirectURL string) bool {
	base, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	target, err := url.Parse(redirectURL)
	if err != nil || target.Scheme != "https" || target.Host != base.Host {
		return false
	}
	return strings.HasPrefix(target.Path, "/snap/v2/vtweb/")
}

// VerifyNotification validates Midtrans' SHA-512 signature_key using the
// provider's documented order_id + status_code + gross_amount + server key
// concatenation. Comparison is constant-time.
func VerifyNotification(serverKey string, n Notification) bool {
	if serverKey == "" || n.OrderID == "" || n.StatusCode == "" || n.GrossAmount == "" || n.SignatureKey == "" {
		return false
	}
	sum := sha512.Sum512([]byte(n.OrderID + n.StatusCode + n.GrossAmount + serverKey))
	expected := sum[:]
	provided, err := hex.DecodeString(strings.TrimSpace(n.SignatureKey))
	if err != nil || len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare(provided, expected) == 1
}

// ParseGrossAmount parses an IDR amount without floating-point rounding.
func ParseGrossAmount(value string) (int64, error) {
	rat, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok || rat.Sign() <= 0 {
		return 0, fmt.Errorf("invalid gross_amount")
	}
	whole, remainder := new(big.Int), new(big.Int)
	whole.QuoRem(rat.Num(), rat.Denom(), remainder)
	if remainder.Sign() != 0 || !whole.IsInt64() {
		return 0, fmt.Errorf("gross_amount must be a positive integer IDR amount")
	}
	return whole.Int64(), nil
}
