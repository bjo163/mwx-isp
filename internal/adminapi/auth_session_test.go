package adminapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// sessionClaims builds the claim set issueToken emits, with a caller-chosen
// expiry so abuse cases can be signed exactly like a real token.
func sessionClaims(operator domain.SysOpr, expires time.Time) jwt.MapClaims {
	now := time.Now()
	return jwt.MapClaims{
		"sub":                fmt.Sprintf("%d", operator.ID),
		"tenant_id":          fmt.Sprintf("%d", operator.TenantID),
		"membership_version": operator.MembershipVersion,
		"username":           operator.Username,
		"role":               operator.Level,
		"exp":                expires.Unix(),
		"iat":                now.Unix(),
		"nbf":                now.Add(-time.Minute).Unix(),
		"iss":                "mwx-isp",
	}
}

func sessionRequest(t *testing.T, handler http.Handler, method, path string, mutate func(*http.Request)) (int, []byte) {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

// TestAdminTokenAbuseIsRejected drives the real router and JWT middleware with
// forged, expired and credential-shuffling tokens. The admin API must accept
// exactly one shape: a header bearer token signed with web.secret and bound to
// an operator whose membership version still matches.
func TestAdminTokenAbuseIsRejected(t *testing.T) {
	handler, db, appCtx := newRoleMatrixServer(t)
	operator := seedMatrixOperator(t, db, 9401, LevelAdmin, "session-admin")
	valid := matrixToken(t, appCtx, operator)
	secret := []byte(appCtx.Config().Web.Secret)

	expired, err := jwt.NewWithClaims(jwt.SigningMethodHS256, sessionClaims(operator, time.Now().Add(-time.Hour))).SignedString(secret)
	require.NoError(t, err)
	wrongKey, err := jwt.NewWithClaims(jwt.SigningMethodHS256, sessionClaims(operator, time.Now().Add(time.Hour))).SignedString([]byte("attacker-controlled-key"))
	require.NoError(t, err)
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, sessionClaims(operator, time.Now().Add(time.Hour))).SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	parts := strings.Split(valid, ".")
	require.Len(t, parts, 3)
	tamperedPayload := base64.RawURLEncoding.EncodeToString([]byte(
		fmt.Sprintf(`{"sub":"%d","tenant_id":"%d","username":"%s","role":"super","exp":%d,"iss":"mwx-isp"}`,
			operator.ID, operator.TenantID, operator.Username, time.Now().Add(time.Hour).Unix())))
	tampered := parts[0] + "." + tamperedPayload + "." + parts[2]

	bearer := func(token string) func(*http.Request) {
		return func(req *http.Request) { req.Header.Set("Authorization", "Bearer "+token) }
	}

	for _, tc := range []struct {
		name   string
		mutate func(*http.Request)
		want   int
	}{
		{name: "valid bearer token", mutate: bearer(valid), want: http.StatusOK},
		{name: "no credentials", mutate: nil, want: http.StatusUnauthorized},
		{name: "expired token", mutate: bearer(expired), want: http.StatusUnauthorized},
		{name: "signed with another key", mutate: bearer(wrongKey), want: http.StatusUnauthorized},
		{name: "alg none", mutate: bearer(unsigned), want: http.StatusUnauthorized},
		{name: "tampered payload", mutate: bearer(tampered), want: http.StatusUnauthorized},
		{name: "wrong auth scheme", mutate: func(req *http.Request) { req.Header.Set("Authorization", "Token "+valid) }, want: http.StatusUnauthorized},
		{name: "token in cookie only", mutate: func(req *http.Request) {
			req.AddCookie(&http.Cookie{Name: "token", Value: valid})   //nolint:gosec // G124: the attacker-supplied cookie is the input under test.
			req.AddCookie(&http.Cookie{Name: "session", Value: valid}) //nolint:gosec // G124: the attacker-supplied cookie is the input under test.
		}, want: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, body := sessionRequest(t, handler, http.MethodGet, "/api/v1/auth/me", tc.mutate)
			assert.Equalf(t, tc.want, status, "%s", body)
			assert.NotContains(t, string(body), string(secret), "the signing secret must never be echoed")
		})
	}

	// A successful read returns account data without credential material: the
	// stored password value is never echoed.
	status, body := sessionRequest(t, handler, http.MethodGet, "/api/v1/auth/me", bearer(valid))
	require.Equalf(t, http.StatusOK, status, "%s", body)
	var self struct {
		Data struct {
			User struct {
				Username string `json:"username"`
				Password string `json:"password"`
			} `json:"user"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &self))
	assert.Equal(t, operator.Username, self.Data.User.Username)
	assert.Empty(t, self.Data.User.Password)
	assert.NotContains(t, string(body), "unused-in-this-test", "the stored credential must not be returned")
}

// TestAdminSessionHasNoServerSideRefreshOrLogout pins the session lifecycle the
// threat model relies on: the token is a 12h absolute bearer credential, there
// is no refresh endpoint and no server-side logout, so a copied token stays
// usable until it expires. Revocation is the operator/membership path tested
// elsewhere (disabled account and membership version bump).
func TestAdminSessionHasNoServerSideRefreshOrLogout(t *testing.T) {
	handler, db, appCtx := newRoleMatrixServer(t)
	operator := seedMatrixOperator(t, db, 9402, LevelAdmin, "session-logout")
	valid := matrixToken(t, appCtx, operator)
	bearer := func(token string) func(*http.Request) {
		return func(req *http.Request) { req.Header.Set("Authorization", "Bearer "+token) }
	}

	// Unmatched /api/v1 paths still pass through the JWT middleware, so an
	// anonymous caller learns nothing about which routes exist.
	for _, path := range []string{"/api/v1/auth/logout", "/api/v1/auth/refresh"} {
		for _, tc := range []struct {
			name   string
			mutate func(*http.Request)
			want   int
		}{
			{name: "anonymous", mutate: nil, want: http.StatusUnauthorized},
			{name: "authenticated", mutate: bearer(valid), want: http.StatusNotFound},
		} {
			status, _ := sessionRequest(t, handler, http.MethodPost, path, tc.mutate)
			assert.Equalf(t, tc.want, status, "%s must not be a registered route (%s)", path, tc.name)
		}
	}

	// "Logout" is client-side storage clearing only: the token keeps working.
	status, body := sessionRequest(t, handler, http.MethodGet, "/api/v1/auth/me", bearer(valid))
	assert.Equalf(t, http.StatusOK, status, "%s", body)

	// Login never sets a cookie; the browser credential is the response body.
	hashed, err := common.HashPassword("session-password")
	require.NoError(t, err)
	operator.Password = hashed
	require.NoError(t, db.Model(&domain.SysOpr{}).Where("id = ?", operator.ID).Update("password", hashed).Error)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"username":"session-logout","password":"session-password","tenant_slug":"default"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equalf(t, http.StatusOK, rec.Code, "%s", rec.Body.String())
	assert.Empty(t, rec.Header().Values("Set-Cookie"), "the admin API must not issue ambient cookie credentials")
	assert.Contains(t, rec.Body.String(), `"token"`)
}

// TestAdminTenantBoundFromSignedToken proves the tenant is taken from the signed
// claims, never from a client-supplied header or query parameter. A token for
// tenant B keeps resolving tenant B even when the request claims tenant 1.
func TestAdminTenantBoundFromSignedToken(t *testing.T) {
	handler, db, appCtx := newRoleMatrixServer(t)
	require.NoError(t, db.Create(&domain.Tenant{
		ID: 2, Name: "Second ISP", Slug: "second", Kind: "isp", Status: "active",
	}).Error)
	// SysOpr.AfterCreate provisions the (tenant 2) membership grant.
	operatorB := domain.SysOpr{
		ID: 9403, TenantID: 2, Username: "session-tenant-b", Password: "unused-in-this-test",
		Level: LevelAdmin, Status: common.ENABLED, MembershipVersion: 1,
	}
	require.NoError(t, db.Create(&operatorB).Error)

	operatorA := seedMatrixOperator(t, db, 9404, LevelAdmin, "session-tenant-a")

	for _, tc := range []struct {
		name     string
		operator domain.SysOpr
		wantID   string
	}{
		{name: "tenant A", operator: operatorA, wantID: "1"},
		{name: "tenant B", operator: operatorB, wantID: "2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			token := matrixToken(t, appCtx, tc.operator)
			status, body := sessionRequest(t, handler, http.MethodGet,
				"/api/v1/auth/me?tenant_id=1&tenant_slug=default", func(req *http.Request) {
					req.Header.Set("Authorization", "Bearer "+token)
					req.Header.Set("X-Tenant-ID", "1")
				})
			require.Equalf(t, http.StatusOK, status, "%s", body)

			var envelope struct {
				Data struct {
					Tenant struct {
						ID string `json:"id"`
					} `json:"tenant"`
					User struct {
						Username string `json:"username"`
					} `json:"user"`
				} `json:"data"`
			}
			require.NoError(t, json.Unmarshal(body, &envelope))
			assert.Equal(t, tc.wantID, envelope.Data.Tenant.ID, "tenant must come from the token claim")
			assert.Equal(t, tc.operator.Username, envelope.Data.User.Username)
		})
	}
}

// TestAdminBrowserStorageContractKeepsBearerHeaderAuth ratchets the frontend
// half of the threat model: the token lives in localStorage and is sent as an
// Authorization header. No cookie credentials are used, which is why the CSRF
// class of attacks against the admin API is not applicable; switching to
// cookies would require a CSRF design and a conscious change to this test.
func TestAdminBrowserStorageContractKeepsBearerHeaderAuth(t *testing.T) {
	root := filepath.Join("..", "..", "web", "src")
	var offenders []string
	require.NoError(t, filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".ts" && ext != ".tsx" {
			return nil
		}
		raw, readErr := os.ReadFile(path) //nolint:gosec // walks the repository source tree
		if readErr != nil {
			return readErr
		}
		source := string(raw)
		for _, marker := range []string{"document.cookie", "withCredentials", "credentials: 'include'", `credentials: "include"`} {
			if strings.Contains(source, marker) {
				offenders = append(offenders, path+" uses "+marker)
			}
		}
		return nil
	}))
	assert.Empty(t, offenders, "cookie-based admin credentials need a CSRF mitigation before use")

	apiClient, err := os.ReadFile(filepath.Join(root, "utils", "apiClient.ts")) //nolint:gosec // G304: fixed file inside the repository.
	require.NoError(t, err)
	assert.Contains(t, string(apiClient), "localStorage.getItem('token')")
	assert.Contains(t, string(apiClient), "`Bearer ${token}`")

	storage, err := os.ReadFile(filepath.Join(root, "utils", "storage.ts")) //nolint:gosec // G304: fixed file inside the repository.
	require.NoError(t, err)
	assert.Contains(t, string(storage), "'token'", "client-side logout must clear the token key")
}
