package adminapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// newUploadRequest builds a multipart request carrying a single "upload" file.
func newUploadRequest(filename string, content []byte) (*http.Request, *httptest.ResponseRecorder) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("upload", filename)
	_, _ = part.Write(content)
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/import", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	return req, rec
}

func TestImportRadiusUsers_JSON(t *testing.T) {
	db := setupTestDB(t)
	appCtx := setupTestApp(t, db)
	profile := createTestProfile(db, "import-profile")

	lines := []map[string]interface{}{
		{"username": "imp_user1", "password": "secret123", "profile_id": profile.ID, "realname": "User One"},
		{"username": "imp_user2", "password": "secret123", "profile_id": profile.ID, "email": "u2@example.com"},
	}
	var content bytes.Buffer
	for _, l := range lines {
		bs, _ := json.Marshal(l)
		content.Write(bs)
		content.WriteByte('\n')
	}

	req, rec := newUploadRequest("users.json", content.Bytes())
	e := setupTestEcho()
	c := CreateTestContext(e, db, req, rec, appCtx)

	require.NoError(t, importRadiusUsers(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, _ := json.Marshal(resp.Data)
	var result ImportUserResult
	require.NoError(t, json.Unmarshal(data, &result))

	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 2, result.Success)
	assert.Equal(t, 0, result.Failed)

	var count int64
	db.Model(&domain.RadiusUser{}).Count(&count)
	assert.Equal(t, int64(2), count)
}

func TestImportRadiusUsers_RadiusClass(t *testing.T) {
	db := setupTestDB(t)
	appCtx := setupTestApp(t, db)
	profile := createTestProfile(db, "import-class-profile")
	profile.RadiusClass = "OU=profile-group"
	require.NoError(t, db.Save(profile).Error)

	lines := []map[string]interface{}{
		{"username": "imp_class_inherit", "password": "secret123", "profile_id": profile.ID},
		{"username": "imp_class_override", "password": "secret123", "profile_id": profile.ID, "radius_class": "OU=user-group"},
	}
	var content bytes.Buffer
	for _, l := range lines {
		bs, _ := json.Marshal(l)
		content.Write(bs)
		content.WriteByte('\n')
	}

	req, rec := newUploadRequest("users.json", content.Bytes())
	e := setupTestEcho()
	c := CreateTestContext(e, db, req, rec, appCtx)

	require.NoError(t, importRadiusUsers(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var inherited domain.RadiusUser
	require.NoError(t, db.Where("username = ?", "imp_class_inherit").First(&inherited).Error)
	assert.Equal(t, "OU=profile-group", inherited.RadiusClass)

	var overridden domain.RadiusUser
	require.NoError(t, db.Where("username = ?", "imp_class_override").First(&overridden).Error)
	assert.Equal(t, "OU=user-group", overridden.RadiusClass)
}

func TestImportRadiusUsers_CSV(t *testing.T) {
	db := setupTestDB(t)
	appCtx := setupTestApp(t, db)
	profile := createTestProfile(db, "import-profile")

	csv := "username,password,profile_id,mobile\n" +
		"csv_user1,secret123," + strconv.FormatInt(profile.ID, 10) + ",13800138000\n" +
		"csv_user2,secret123," + strconv.FormatInt(profile.ID, 10) + ",13800138001\n"

	req, rec := newUploadRequest("users.csv", []byte(csv))
	e := setupTestEcho()
	c := CreateTestContext(e, db, req, rec, appCtx)

	require.NoError(t, importRadiusUsers(c))
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, _ := json.Marshal(resp.Data)
	var result ImportUserResult
	require.NoError(t, json.Unmarshal(data, &result))

	assert.Equal(t, 2, result.Success)
}

func TestImportRadiusUsers_Errors(t *testing.T) {
	db := setupTestDB(t)
	appCtx := setupTestApp(t, db)
	profile := createTestProfile(db, "import-profile")
	createTestUser(db, "dup_user", profile.ID)

	lines := []map[string]interface{}{
		{"username": "", "password": "secret123", "profile_id": profile.ID},         // missing username
		{"username": "no_pass", "profile_id": profile.ID},                           // missing password
		{"username": "bad_profile", "password": "secret123", "profile_id": 999999},  // missing profile
		{"username": "dup_user", "password": "secret123", "profile_id": profile.ID}, // duplicate
		{"username": "ok_user", "password": "secret123", "profile_id": profile.ID},  // valid
	}
	var content bytes.Buffer
	for _, l := range lines {
		bs, _ := json.Marshal(l)
		content.Write(bs)
		content.WriteByte('\n')
	}

	req, rec := newUploadRequest("users.json", content.Bytes())
	e := setupTestEcho()
	c := CreateTestContext(e, db, req, rec, appCtx)

	require.NoError(t, importRadiusUsers(c))

	var resp Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, _ := json.Marshal(resp.Data)
	var result ImportUserResult
	require.NoError(t, json.Unmarshal(data, &result))

	assert.Equal(t, 5, result.Total)
	// One bad row rejects the whole payload: nothing is written, so the valid
	// row is not imported either.
	assert.Equal(t, 0, result.Success)
	assert.Equal(t, 4, result.Failed)
	assert.Len(t, result.Errors, 4)

	var count int64
	db.Model(&domain.RadiusUser{}).Count(&count)
	assert.Equal(t, int64(1), count, "only the pre-existing dup_user may remain")
	var validRows int64
	require.NoError(t, db.Model(&domain.RadiusUser{}).Where("username = ?", "ok_user").Count(&validRows).Error)
	assert.Equal(t, int64(0), validRows, "the valid row must not be written when the file is rejected")
}

// importJSON builds a newline-delimited JSON payload from the given rows.
func importJSON(t *testing.T, lines []map[string]interface{}) []byte {
	t.Helper()
	var content bytes.Buffer
	for _, l := range lines {
		bs, err := json.Marshal(l)
		require.NoError(t, err)
		content.Write(bs)
		content.WriteByte('\n')
	}
	return content.Bytes()
}

// runImport executes the handler against a JSON payload and decodes the result.
func runImport(t *testing.T, db *gorm.DB, content []byte) ImportUserResult {
	t.Helper()
	appCtx := setupTestApp(t, db)
	req, rec := newUploadRequest("users.json", content)
	e := setupTestEcho()
	c := CreateTestContext(e, db, req, rec, appCtx)
	require.NoError(t, importRadiusUsers(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var resp Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	data, err := json.Marshal(resp.Data)
	require.NoError(t, err)
	var result ImportUserResult
	require.NoError(t, json.Unmarshal(data, &result))
	return result
}

// TestImportRadiusUsers_RejectsDuplicateRowsWithinFile pins in-file duplicate
// detection. The duplicate check used to rely on the previous insert having
// committed, which only worked because the importer wrote row by row.
func TestImportRadiusUsers_RejectsDuplicateRowsWithinFile(t *testing.T) {
	db := setupTestDB(t)
	profile := createTestProfile(db, "import-dup-profile")

	content := importJSON(t, []map[string]interface{}{
		{"username": "same_user", "password": "secret123", "profile_id": profile.ID},
		{"username": "same_user", "password": "secret456", "profile_id": profile.ID},
	})
	result := runImport(t, db, content)

	assert.Equal(t, 2, result.Total)
	assert.Equal(t, 0, result.Success)
	assert.Equal(t, 1, result.Failed)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, 2, result.Errors[0].Row)
	assert.Equal(t, "same_user", result.Errors[0].Username)
	assert.Contains(t, result.Errors[0].Message, "duplicates row 1")

	var count int64
	db.Model(&domain.RadiusUser{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

// TestImportRadiusUsers_RejectsUnusableValues covers rows the old importer
// either accepted silently or reported with a misleading message.
func TestImportRadiusUsers_RejectsUnusableValues(t *testing.T) {
	db := setupTestDB(t)
	profile := createTestProfile(db, "import-values-profile")

	cases := []struct {
		name    string
		row     map[string]interface{}
		message string
	}{
		{
			name:    "non numeric profile id",
			row:     map[string]interface{}{"username": "bad_profile", "password": "secret123", "profile_id": "abc"},
			message: "Billing profile id must be a positive integer",
		},
		{
			name:    "unknown status would import an enabled user",
			row:     map[string]interface{}{"username": "bad_status", "password": "secret123", "profile_id": profile.ID, "status": "nonaktif"},
			message: "Status must be enabled or disabled",
		},
		{
			name:    "non numeric node id",
			row:     map[string]interface{}{"username": "bad_node", "password": "secret123", "profile_id": profile.ID, "node_id": "edge-1"},
			message: "Network node id must be a positive integer",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runImport(t, db, importJSON(t, []map[string]interface{}{tc.row}))

			assert.Equal(t, 1, result.Total)
			assert.Equal(t, 0, result.Success)
			assert.Equal(t, 1, result.Failed)
			require.Len(t, result.Errors, 1)
			assert.Contains(t, result.Errors[0].Message, tc.message)

			var count int64
			db.Model(&domain.RadiusUser{}).Count(&count)
			assert.Equal(t, int64(0), count)
		})
	}
}

// TestImportRadiusUsers_AcceptsDisabledStatus keeps the one non-default status
// the auth checker understands importable.
func TestImportRadiusUsers_AcceptsDisabledStatus(t *testing.T) {
	db := setupTestDB(t)
	profile := createTestProfile(db, "import-disabled-profile")

	result := runImport(t, db, importJSON(t, []map[string]interface{}{
		{"username": "disabled_user", "password": "secret123", "profile_id": profile.ID, "status": "Disabled"},
	}))

	assert.Equal(t, 1, result.Total)
	assert.Equal(t, 1, result.Success)
	assert.Equal(t, 0, result.Failed)

	var user domain.RadiusUser
	require.NoError(t, db.Where("username = ?", "disabled_user").First(&user).Error)
	assert.Equal(t, common.DISABLED, user.Status)
}
