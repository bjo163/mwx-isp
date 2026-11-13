package midtrans

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCreateCheckoutUsesServerSideCredentialsAndInvoiceAmount(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		require.Equal(t, "Basic "+base64.StdEncoding.EncodeToString([]byte("secret:")), r.Header.Get("Authorization"))
		var payload createRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, "MWX-1-2-3", payload.TransactionDetails.OrderID)
		require.Equal(t, int64(150000), payload.TransactionDetails.GrossAmount)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(createResponse{Token: "snap-token", RedirectURL: "https://" + r.Host + "/snap/v2/vtweb/test-token"})
	}))
	defer server.Close()

	client, err := New(Config{ServerKey: "secret", ClientKey: "client", BaseURL: server.URL, Timeout: time.Second}, server.Client())
	require.NoError(t, err)
	checkout, err := client.CreateCheckout(context.Background(), "MWX-1-2-3", 150000, Customer{Name: "Customer"})
	require.NoError(t, err)
	require.Equal(t, "snap-token", checkout.Token)
	require.Contains(t, checkout.RedirectURL, "/snap/v2/vtweb/")
}

func TestCreateCheckoutRejectsUntrustedRedirect(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(createResponse{Token: "token", RedirectURL: "https://evil.example/snap/v2/vtweb/token"})
	}))
	defer server.Close()
	client, err := New(Config{ServerKey: "secret", ClientKey: "client", BaseURL: server.URL}, server.Client())
	require.NoError(t, err)
	_, err = client.CreateCheckout(context.Background(), "MWX-1", 1000, Customer{})
	require.Error(t, err)
}

func TestVerifyNotification(t *testing.T) {
	n := Notification{OrderID: "MWX-1", StatusCode: "200", GrossAmount: "12000.00"}
	sum := sha512.Sum512([]byte(n.OrderID + n.StatusCode + n.GrossAmount + "server-key"))
	n.SignatureKey = hex.EncodeToString(sum[:])
	require.True(t, VerifyNotification("server-key", n))
	n.GrossAmount = "12001.00"
	require.False(t, VerifyNotification("server-key", n))
}

func TestParseGrossAmount(t *testing.T) {
	amount, err := ParseGrossAmount("125000.00")
	require.NoError(t, err)
	require.Equal(t, int64(125000), amount)
	_, err = ParseGrossAmount("125000.50")
	require.Error(t, err)
	_, err = ParseGrossAmount("not-money")
	require.Error(t, err)
}
