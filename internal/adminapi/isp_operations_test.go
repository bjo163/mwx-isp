package adminapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/pkg/common"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"layeh.com/radius"
)

func TestISPOperations(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(
		&domain.HotspotBatch{},
		&domain.HotspotVoucher{},
		&domain.InternetPackage{},
		&domain.RadiusProfile{},
		&domain.DocumentSequence{},
		&domain.IPAMPool{},
		&domain.TroubleTicket{},
		&domain.RadiusUser{},
		&domain.Customer{},
		&domain.NotificationSettings{},
		&domain.NotificationOutbox{},
	))
	profile := domain.RadiusProfile{Name: "Hotspot 10 Mbps", Status: "enabled", UpRate: 10000, DownRate: 10000, ActiveNum: 1}
	require.NoError(t, db.Create(&profile).Error)
	pkg := domain.InternetPackage{Code: "HOT-10", Name: "Hotspot 10 Mbps", Status: "active", RadiusProfileID: profile.ID}
	require.NoError(t, db.Create(&pkg).Error)

	// 1. Test Generate Vouchers
	genPayload := []byte(`{
		"name": "Warkop 5k",
		"package_id": "` + strconv.FormatInt(pkg.ID, 10) + `",
		"quantity": 5,
		"price": 5000,
		"validity_seconds": 10800,
		"quota_mb": 0,
		"prefix": "WRK",
		"code_length": 5,
		"same_user_pass": true
	}`)
	reqGen := httptest.NewRequest(http.MethodPost, "/api/v1/isp/vouchers/generate", bytes.NewReader(genPayload))
	reqGen.Header.Set("Content-Type", "application/json")
	recGen := httptest.NewRecorder()
	cGen := CreateTestContext(e, db, reqGen, recGen, appCtx)

	err := generateVouchers(cGen)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, recGen.Code)

	var vCount int64
	require.NoError(t, db.Model(&domain.HotspotVoucher{}).Count(&vCount).Error)
	assert.Equal(t, int64(5), vCount)
	var batchCount int64
	require.NoError(t, db.Model(&domain.HotspotBatch{}).Count(&batchCount).Error)
	assert.Equal(t, int64(1), batchCount)
	var generatedBatch domain.HotspotBatch
	require.NoError(t, db.First(&generatedBatch).Error)
	assert.Equal(t, "superadmin", generatedBatch.CreatedBy)
	var radiusCount int64
	require.NoError(t, db.Model(&domain.RadiusUser{}).Where("profile_id = ?", profile.ID).Count(&radiusCount).Error)
	assert.Equal(t, int64(5), radiusCount)
	var generatedUser domain.RadiusUser
	require.NoError(t, db.Where("profile_id = ?", profile.ID).First(&generatedUser).Error)
	assert.Equal(t, "enabled", generatedUser.Status)
	assert.True(t, generatedUser.ExpireTime.After(time.Now()))
	var generatedVoucher domain.HotspotVoucher
	require.NoError(t, db.Where("code = ?", generatedUser.Username).First(&generatedVoucher).Error)
	assert.Equal(t, generatedUser.ID, generatedVoucher.RadiusUserID)

	// 2. Test List Vouchers
	reqListV := httptest.NewRequest(http.MethodGet, "/api/v1/isp/vouchers", nil)
	recListV := httptest.NewRecorder()
	cListV := CreateTestContext(e, db, reqListV, recListV, appCtx)

	err = listVouchers(cListV)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, recListV.Code)

	// 3. Test IPAM Pool Creation & Listing
	ipamPayload := []byte(`{
		"name": "PPPoE CGNAT Pool 1",
		"cidr": "100.64.0.0/22",
		"pool_type": "cgnat",
		"gateway": "100.64.0.1",
		"dns_primary": "1.1.1.1"
	}`)
	reqIPAM := httptest.NewRequest(http.MethodPost, "/api/v1/network/ipam/pools", bytes.NewReader(ipamPayload))
	reqIPAM.Header.Set("Content-Type", "application/json")
	recIPAM := httptest.NewRecorder()
	cIPAM := CreateTestContext(e, db, reqIPAM, recIPAM, appCtx)

	err = createIPAMPool(cIPAM)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, recIPAM.Code)

	reqListIPAM := httptest.NewRequest(http.MethodGet, "/api/v1/network/ipam/pools", nil)
	recListIPAM := httptest.NewRecorder()
	cListIPAM := CreateTestContext(e, db, reqListIPAM, recListIPAM, appCtx)

	err = listIPAMPools(cListIPAM)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, recListIPAM.Code)

	// 4. Test Trouble Ticket Creation & Update
	customer := domain.Customer{
		Name:    "Pak Joko",
		Phone:   "081234567890",
		Address: "Jl. Merdeka No. 45",
	}
	require.NoError(t, db.Create(&customer).Error)

	ticketPayload := []byte(`{
		"customer_id": "` + strconv.FormatInt(customer.ID, 10) + `",
		"subject": "Kabel FO Terputus / LOS Merah",
		"category": "los_red",
		"priority": "urgent",
		"assigned_technician": "Budi Lapangan",
		"technician_phone": "081987654321"
	}`)
	reqTicket := httptest.NewRequest(http.MethodPost, "/api/v1/isp/tickets", bytes.NewReader(ticketPayload))
	reqTicket.Header.Set("Content-Type", "application/json")
	recTicket := httptest.NewRecorder()
	cTicket := CreateTestContext(e, db, reqTicket, recTicket, appCtx)

	err = createTroubleTicket(cTicket)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, recTicket.Code)

	var ticketRes struct {
		Data domain.TroubleTicket `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recTicket.Body.Bytes(), &ticketRes))
	assert.Equal(t, "urgent", ticketRes.Data.Priority)
	assert.Equal(t, "open", ticketRes.Data.Status)

	// Dispatch only claims success when the outbox actually queued a message,
	// so enable the event for one allowlisted recipient first.
	acknowledged := time.Now()
	require.NoError(t, db.Create(&domain.NotificationSettings{
		TenantID:           domain.DefaultTenantID,
		WhatsAppEnabled:    true,
		RiskAcknowledgedAt: &acknowledged,
		RecipientsJSON:     `["+6281234567890"]`,
		EventsJSON:         `["ticket.assigned"]`,
	}).Error)

	// Dispatch ticket WhatsApp
	reqDispatch := httptest.NewRequest(http.MethodPost, "/api/v1/isp/tickets/"+strconv.FormatInt(ticketRes.Data.ID, 10)+"/dispatch", nil)
	recDispatch := httptest.NewRecorder()
	cDispatch := CreateTestContext(e, db, reqDispatch, recDispatch, appCtx)
	cDispatch.SetParamNames("id")
	cDispatch.SetParamValues(strconv.FormatInt(ticketRes.Data.ID, 10))

	err = dispatchTicketWhatsApp(cDispatch)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, recDispatch.Code, recDispatch.Body.String())

	var dispatchRes struct {
		Data struct {
			Queued int `json:"queued"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recDispatch.Body.Bytes(), &dispatchRes))
	assert.Equal(t, 1, dispatchRes.Data.Queued)
	var outbox domain.NotificationOutbox
	require.NoError(t, db.First(&outbox).Error)
	assert.Equal(t, "ticket.assigned", outbox.EventType)
	assert.Equal(t, "6281234567890", outbox.Recipient)
	assert.Contains(t, outbox.Body, "Budi Lapangan")

	// 5. Test Live Ping Tool
	pingPayload := []byte(`{"host": "127.0.0.1", "count": 2}`)
	reqPing := httptest.NewRequest(http.MethodPost, "/api/v1/network/diagnostics/ping", bytes.NewReader(pingPayload))
	reqPing.Header.Set("Content-Type", "application/json")
	recPing := httptest.NewRecorder()
	cPing := CreateTestContext(e, db, reqPing, recPing, appCtx)

	err = runLivePing(cPing)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, recPing.Code)
}

func TestVoucherGenerationRequiresUsablePackageAndRollsBack(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.HotspotBatch{}, &domain.HotspotVoucher{}, &domain.InternetPackage{}, &domain.RadiusProfile{}, &domain.RadiusUser{}, &domain.DocumentSequence{}))
	pkg := domain.InternetPackage{Code: "NO-PROFILE", Name: "Missing profile", Status: "active"}
	require.NoError(t, db.Create(&pkg).Error)
	body := []byte(`{"name":"Invalid","package_id":"` + strconv.FormatInt(pkg.ID, 10) + `","quantity":2}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/isp/vouchers/generate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	require.NoError(t, generateVouchers(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	for _, model := range []any{&domain.HotspotBatch{}, &domain.HotspotVoucher{}, &domain.RadiusUser{}, &domain.DocumentSequence{}} {
		var count int64
		require.NoError(t, db.Model(model).Count(&count).Error)
		assert.Zero(t, count)
	}
	profile := domain.RadiusProfile{Name: "Usable profile", Status: "enabled"}
	require.NoError(t, db.Create(&profile).Error)
	pkg.RadiusProfileID = profile.ID
	require.NoError(t, db.Save(&pkg).Error)
	body = []byte(`{"name":"Quota claim","package_id":"` + strconv.FormatInt(pkg.ID, 10) + `","quantity":1,"quota_mb":256}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/isp/vouchers/generate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	c = CreateTestContext(e, db, req, rec, appCtx)
	require.NoError(t, generateVouchers(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var batches int64
	require.NoError(t, db.Model(&domain.HotspotBatch{}).Count(&batches).Error)
	assert.Zero(t, batches)

	const callbackName = "test:fail_radius_user_create"
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == (domain.RadiusUser{}).TableName() {
			_ = tx.AddError(errors.New("injected RADIUS credential insert failure"))
		}
	}))
	body = []byte(`{"name":"Rollback","package_id":"` + strconv.FormatInt(pkg.ID, 10) + `","quantity":2}`)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/isp/vouchers/generate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	c = CreateTestContext(e, db, req, rec, appCtx)
	require.NoError(t, generateVouchers(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	require.NoError(t, db.Callback().Create().Remove(callbackName))
	for _, model := range []any{&domain.HotspotBatch{}, &domain.HotspotVoucher{}, &domain.RadiusUser{}, &domain.DocumentSequence{}} {
		var count int64
		require.NoError(t, db.Model(model).Count(&count).Error)
		assert.Zero(t, count, "failed credential creation must roll back every generated record")
	}
}

func TestVoucherBatchNumbersUseAtomicDailySequence(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.HotspotBatch{}, &domain.HotspotVoucher{}, &domain.InternetPackage{}, &domain.RadiusProfile{}, &domain.RadiusUser{}, &domain.DocumentSequence{}))
	profile := domain.RadiusProfile{Name: "Voucher profile", Status: "enabled"}
	require.NoError(t, db.Create(&profile).Error)
	pkg := domain.InternetPackage{Code: "BATCH-TEST", Name: "Batch test", Status: "active", RadiusProfileID: profile.ID}
	require.NoError(t, db.Create(&pkg).Error)
	for i := 0; i < 2; i++ {
		body := []byte(`{"package_id":"` + strconv.FormatInt(pkg.ID, 10) + `","quantity":3,"code_length":8}`)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/isp/vouchers/generate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		ctx := CreateTestContext(e, db, req, rec, appCtx)
		require.NoError(t, generateVouchers(ctx))
		assert.Equal(t, http.StatusOK, rec.Code)
	}
	var batches []domain.HotspotBatch
	require.NoError(t, db.Order("id ASC").Find(&batches).Error)
	require.Len(t, batches, 2)
	assert.NotEqual(t, batches[0].BatchNo, batches[1].BatchNo)
	assert.Equal(t, "BATCH-"+time.Now().Format("060102")+"-000001", batches[0].BatchNo)
	assert.Equal(t, "BATCH-"+time.Now().Format("060102")+"-000002", batches[1].BatchNo)
}

func TestVoucherBatchPaginationSupportsFullPrint(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.HotspotBatch{}, &domain.HotspotVoucher{}, &domain.InternetPackage{}, &domain.RadiusProfile{}, &domain.RadiusUser{}, &domain.DocumentSequence{}))
	profile := domain.RadiusProfile{Name: "Print profile", Status: "enabled"}
	require.NoError(t, db.Create(&profile).Error)
	pkg := domain.InternetPackage{Code: "PRINT-250", Name: "Print 250", Status: "active", RadiusProfileID: profile.ID}
	require.NoError(t, db.Create(&pkg).Error)

	body := []byte(`{"package_id":"` + strconv.FormatInt(pkg.ID, 10) + `","quantity":250,"code_length":8}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/isp/vouchers/generate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	require.NoError(t, generateVouchers(CreateTestContext(e, db, req, rec, appCtx)))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var batch domain.HotspotBatch
	require.NoError(t, db.Order("id DESC").First(&batch).Error)
	require.EqualValues(t, 250, batch.Quantity)

	// Mirror the print dialog: walk every page of the batch instead of trusting
	// a single page capped at 100 rows.
	seen := make(map[string]struct{}, 250)
	var reportedTotal int64
	for page := 1; ; page++ {
		listReq := httptest.NewRequest(http.MethodGet,
			fmt.Sprintf("/api/v1/isp/vouchers?batch_id=%d&page=%d&perPage=100", batch.ID, page), nil)
		listRec := httptest.NewRecorder()
		require.NoError(t, listVouchers(CreateTestContext(e, db, listReq, listRec, appCtx)))
		require.Equal(t, http.StatusOK, listRec.Code, listRec.Body.String())
		var payload struct {
			Data []domain.HotspotVoucher `json:"data"`
			Meta struct {
				Total    int64 `json:"total"`
				Page     int   `json:"page"`
				PageSize int   `json:"pageSize"`
			} `json:"meta"`
		}
		require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &payload))
		assert.Equal(t, page, payload.Meta.Page)
		assert.Equal(t, 100, payload.Meta.PageSize)
		reportedTotal = payload.Meta.Total
		if len(payload.Data) == 0 {
			break
		}
		for _, voucher := range payload.Data {
			assert.Equal(t, batch.ID, voucher.BatchID)
			seen[voucher.Code] = struct{}{}
		}
		if int64(len(seen)) >= reportedTotal {
			break
		}
	}
	assert.EqualValues(t, 250, reportedTotal, "server must report the full batch size")
	assert.Len(t, seen, 250, "printing must see 250 unique voucher codes across pages")
}

func TestDeleteVoucherBatchReportsMissingAndRemovesCredentials(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.HotspotBatch{}, &domain.HotspotVoucher{}, &domain.RadiusUser{}))
	batch := domain.HotspotBatch{BatchNo: "BATCH-DELETE-1", Quantity: 1}
	require.NoError(t, db.Create(&batch).Error)
	user := domain.RadiusUser{Username: "DELETE-ME", Password: "secret", Status: "enabled"}
	require.NoError(t, db.Create(&user).Error)
	voucher := domain.HotspotVoucher{BatchID: batch.ID, Code: user.Username, Password: user.Password, RadiusUserID: user.ID, Status: "active"}
	require.NoError(t, db.Create(&voucher).Error)

	missingReq := httptest.NewRequest(http.MethodDelete, "/api/v1/isp/vouchers/batches/999", nil)
	missingRec := httptest.NewRecorder()
	missingCtx := CreateTestContext(e, db, missingReq, missingRec, appCtx)
	missingCtx.SetParamNames("id")
	missingCtx.SetParamValues("999")
	require.NoError(t, deleteVoucherBatch(missingCtx))
	assert.Equal(t, http.StatusNotFound, missingRec.Code)

	const deleteCallbackName = "test:fail_radius_user_delete"
	require.NoError(t, db.Callback().Delete().Before("gorm:delete").Register(deleteCallbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == (domain.RadiusUser{}).TableName() {
			_ = tx.AddError(errors.New("injected RADIUS credential delete failure"))
		}
	}))
	failingReq := httptest.NewRequest(http.MethodDelete, "/api/v1/isp/vouchers/batches/"+strconv.FormatInt(batch.ID, 10), nil)
	failingRec := httptest.NewRecorder()
	failingCtx := CreateTestContext(e, db, failingReq, failingRec, appCtx)
	failingCtx.SetParamNames("id")
	failingCtx.SetParamValues(strconv.FormatInt(batch.ID, 10))
	require.NoError(t, deleteVoucherBatch(failingCtx))
	assert.Equal(t, http.StatusInternalServerError, failingRec.Code)
	require.NoError(t, db.Callback().Delete().Remove(deleteCallbackName))
	for _, model := range []any{&domain.HotspotBatch{}, &domain.HotspotVoucher{}, &domain.RadiusUser{}} {
		var count int64
		require.NoError(t, db.Model(model).Count(&count).Error)
		assert.EqualValues(t, 1, count, "failed delete must roll back every record")
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/isp/vouchers/batches/"+strconv.FormatInt(batch.ID, 10), nil)
	rec := httptest.NewRecorder()
	ctx := CreateTestContext(e, db, req, rec, appCtx)
	ctx.SetParamNames("id")
	ctx.SetParamValues(strconv.FormatInt(batch.ID, 10))
	require.NoError(t, deleteVoucherBatch(ctx))
	assert.Equal(t, http.StatusOK, rec.Code)
	for _, model := range []any{&domain.HotspotBatch{}, &domain.HotspotVoucher{}, &domain.RadiusUser{}} {
		var count int64
		require.NoError(t, db.Model(model).Count(&count).Error)
		assert.Zero(t, count)
	}
}

func TestODPAndFlappingOperations(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(
		&domain.ODP{},
		&domain.Customer{},
		&domain.Subscription{},
		&domain.RadiusUser{},
		&domain.RadiusAccounting{},
		&domain.TroubleTicket{},
		&domain.DocumentSequence{},
	))

	// 1. Create ODP
	odpPayload := []byte(`{
		"code": "ODP-KNG-001",
		"name": "ODP Kuningan Barat 01",
		"zone": "Kuningan",
		"olt_name": "OLT-ZTE-CORE-01",
		"pon_port": "gpon-olt_1/1/3",
		"total_ports": 16,
		"optical_loss": -18.5,
		"status": "active",
		"latitude": -6.2297,
		"longitude": 106.8294
	}`)
	reqODP := httptest.NewRequest(http.MethodPost, "/api/v1/network/odp", bytes.NewReader(odpPayload))
	reqODP.Header.Set("Content-Type", "application/json")
	recODP := httptest.NewRecorder()
	cODP := CreateTestContext(e, db, reqODP, recODP, appCtx)
	require.NoError(t, createODP(cODP))
	assert.Equal(t, http.StatusOK, recODP.Code)

	// 2. List ODPs
	reqListODP := httptest.NewRequest(http.MethodGet, "/api/v1/network/odp", nil)
	recListODP := httptest.NewRecorder()
	cListODP := CreateTestContext(e, db, reqListODP, recListODP, appCtx)
	require.NoError(t, listODPs(cListODP))
	assert.Equal(t, http.StatusOK, recListODP.Code)

	// 2b. Update ODP - clearing notes/address and setting coordinates to 0
	var createdODP domain.ODP
	require.NoError(t, db.Where("code = ?", "ODP-KNG-001").First(&createdODP).Error)

	updatePayload := []byte(`{
		"address": "",
		"notes": "",
		"latitude": 0,
		"longitude": 0,
		"status": "maintenance"
	}`)
	reqUpdateODP := httptest.NewRequest(http.MethodPut, "/api/v1/network/odp/"+strconv.FormatInt(createdODP.ID, 10), bytes.NewReader(updatePayload))
	reqUpdateODP.Header.Set("Content-Type", "application/json")
	recUpdateODP := httptest.NewRecorder()
	cUpdateODP := CreateTestContext(e, db, reqUpdateODP, recUpdateODP, appCtx)
	cUpdateODP.SetParamNames("id")
	cUpdateODP.SetParamValues(strconv.FormatInt(createdODP.ID, 10))
	require.NoError(t, updateODP(cUpdateODP))
	assert.Equal(t, http.StatusOK, recUpdateODP.Code)

	var refreshedODP domain.ODP
	require.NoError(t, db.First(&refreshedODP, createdODP.ID).Error)
	assert.Equal(t, "", refreshedODP.Address)
	assert.Equal(t, "", refreshedODP.Notes)
	assert.Equal(t, float64(0), refreshedODP.Latitude)
	assert.Equal(t, float64(0), refreshedODP.Longitude)
	assert.Equal(t, "maintenance", refreshedODP.Status)
	assert.Equal(t, "ODP Kuningan Barat 01", refreshedODP.Name, "absent name field should be preserved")

	// 3. Flapping Detection and Auto-Ticket
	reqFlap := httptest.NewRequest(http.MethodGet, "/api/v1/network/diagnostics/flapping", nil)
	recFlap := httptest.NewRecorder()
	cFlap := CreateTestContext(e, db, reqFlap, recFlap, appCtx)
	require.NoError(t, detectFlappingSubscribers(cFlap))
	assert.Equal(t, http.StatusOK, recFlap.Code)

	// 4. Auto Ticket for Flapping
	require.NoError(t, db.Create(&domain.RadiusUser{Username: "customer_flap_01", Password: "test-secret", Status: "enabled"}).Error)
	ticketPayload := []byte(`{
		"username": "customer_flap_01",
		"customer_no": "CUST-0099",
		"description": "Redaman dropcore drop ke -29 dBm"
	}`)
	reqTicket := httptest.NewRequest(http.MethodPost, "/api/v1/network/diagnostics/flapping/auto-ticket", bytes.NewReader(ticketPayload))
	reqTicket.Header.Set("Content-Type", "application/json")
	recTicket := httptest.NewRecorder()
	cTicket := CreateTestContext(e, db, reqTicket, recTicket, appCtx)
	require.NoError(t, createFlappingTicket(cTicket))
	assert.Equal(t, http.StatusOK, recTicket.Code)
}

func TestAuditIPHistoryValidationAndTemporalFiltering(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.RadiusOnline{}, &domain.RadiusAccounting{}))

	baseTime := time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC)
	// Active session started at 08:00
	require.NoError(t, db.Create(&domain.RadiusOnline{
		Username:      "user_current",
		FramedIpaddr:  "100.64.10.5",
		AcctSessionId: "SESS-001",
		AcctStartTime: baseTime,
		LastUpdate:    baseTime.Add(30 * time.Minute),
	}).Error)

	// Historical session yesterday
	yesterdayStart := baseTime.Add(-24 * time.Hour)
	yesterdayStop := yesterdayStart.Add(2 * time.Hour)
	require.NoError(t, db.Create(&domain.RadiusAccounting{
		Username:        "user_yesterday",
		FramedIpaddr:    "100.64.10.5",
		AcctSessionId:   "SESS-000",
		AcctStartTime:   yesterdayStart,
		AcctStopTime:    yesterdayStop,
		AcctSessionTime: 7200,
	}).Error)

	// 1. Invalid timestamp returns 400
	reqInvalid := httptest.NewRequest(http.MethodGet, "/api/v1/network/ipam/audit?ip=100.64.10.5&at=invalid-time", nil)
	recInvalid := httptest.NewRecorder()
	ctxInvalid := CreateTestContext(e, db, reqInvalid, recInvalid, appCtx)
	require.NoError(t, auditIPHistory(ctxInvalid))
	assert.Equal(t, http.StatusBadRequest, recInvalid.Code)

	// 2. Query yesterday time returns only user_yesterday, NOT user_current
	targetYesterday := yesterdayStart.Add(1 * time.Hour).Format(time.RFC3339)
	reqPast := httptest.NewRequest(http.MethodGet, "/api/v1/network/ipam/audit?ip=100.64.10.5&at="+targetYesterday, nil)
	recPast := httptest.NewRecorder()
	ctxPast := CreateTestContext(e, db, reqPast, recPast, appCtx)
	require.NoError(t, auditIPHistory(ctxPast))
	assert.Equal(t, http.StatusOK, recPast.Code)

	var resPast struct {
		Data struct {
			QueriedIP  string           `json:"queried_ip"`
			TotalFound int              `json:"total_found"`
			Results    []map[string]any `json:"results"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recPast.Body.Bytes(), &resPast))
	assert.Equal(t, 1, resPast.Data.TotalFound)
	require.Len(t, resPast.Data.Results, 1)
	assert.Equal(t, "user_yesterday", resPast.Data.Results[0]["username"])
	assert.Equal(t, "historical_accounting", resPast.Data.Results[0]["type"])

	// 3. Query current time returns user_current
	targetNow := baseTime.Add(15 * time.Minute).Format(time.RFC3339)
	reqNow := httptest.NewRequest(http.MethodGet, "/api/v1/network/ipam/audit?ip=100.64.10.5&at="+targetNow, nil)
	recNow := httptest.NewRecorder()
	ctxNow := CreateTestContext(e, db, reqNow, recNow, appCtx)
	require.NoError(t, auditIPHistory(ctxNow))
	assert.Equal(t, http.StatusOK, recNow.Code)

	var resNow struct {
		Data struct {
			TotalFound int              `json:"total_found"`
			Results    []map[string]any `json:"results"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recNow.Body.Bytes(), &resNow))
	assert.Equal(t, 1, resNow.Data.TotalFound)
	assert.Equal(t, "user_current", resNow.Data.Results[0]["username"])
	assert.Equal(t, "live_online", resNow.Data.Results[0]["type"])
}

func TestLivePingProbeIPv4AndIPv6Handling(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)

	// 1. IPv4 loopback with custom port
	bodyIPv4 := []byte(`{"host": "127.0.0.1:80", "count": 1}`)
	reqIPv4 := httptest.NewRequest(http.MethodPost, "/api/v1/network/diagnostics/ping", bytes.NewReader(bodyIPv4))
	reqIPv4.Header.Set("Content-Type", "application/json")
	recIPv4 := httptest.NewRecorder()
	ctxIPv4 := CreateTestContext(e, db, reqIPv4, recIPv4, appCtx)
	require.NoError(t, runLivePing(ctxIPv4))
	assert.Equal(t, http.StatusOK, recIPv4.Code)

	var resIPv4 struct {
		Data struct {
			Host        string `json:"host"`
			TargetAddr  string `json:"target_addr"`
			ProbeMethod string `json:"probe_method"`
			Port        string `json:"port"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recIPv4.Body.Bytes(), &resIPv4))
	assert.Equal(t, "127.0.0.1:80", resIPv4.Data.TargetAddr)
	assert.Equal(t, "tcp_syn", resIPv4.Data.ProbeMethod)
	assert.Equal(t, "80", resIPv4.Data.Port)

	// 2. IPv6 loopback without port
	bodyIPv6 := []byte(`{"host": "::1", "count": 1}`)
	reqIPv6 := httptest.NewRequest(http.MethodPost, "/api/v1/network/diagnostics/ping", bytes.NewReader(bodyIPv6))
	reqIPv6.Header.Set("Content-Type", "application/json")
	recIPv6 := httptest.NewRecorder()
	ctxIPv6 := CreateTestContext(e, db, reqIPv6, recIPv6, appCtx)
	require.NoError(t, runLivePing(ctxIPv6))
	assert.Equal(t, http.StatusOK, recIPv6.Code)

	var resIPv6 struct {
		Data struct {
			TargetAddr  string `json:"target_addr"`
			ProbeMethod string `json:"probe_method"`
			Port        string `json:"port"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recIPv6.Body.Bytes(), &resIPv6))
	assert.Equal(t, "[::1]:80", resIPv6.Data.TargetAddr)
	assert.Equal(t, "80", resIPv6.Data.Port)
}

func newVoucherRevokeCtx(t *testing.T, db *gorm.DB, appCtx app.AppContext, id int64) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := setupTestEcho()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/isp/vouchers/"+strconv.FormatInt(id, 10)+"/revoke", nil)
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	c.SetParamNames("id")
	c.SetParamValues(strconv.FormatInt(id, 10))
	return c, rec
}

type voucherRevokeBody struct {
	Data struct {
		Status               string `json:"status"`
		AlreadyRevoked       bool   `json:"already_revoked"`
		DisconnectedSessions int    `json:"disconnected_sessions"`
		FailedDisconnects    int    `json:"failed_disconnects"`
	} `json:"data"`
}

func TestRevokeVoucherDisablesCredentialAndDisconnectsLiveSessions(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.HotspotBatch{}, &domain.HotspotVoucher{}))
	appCtx := setupTestApp(t, db)

	nas := newFakeCoANAS(t, coaTestSecret, radius.CodeDisconnectACK, 0)
	nasRow := &domain.NetNas{
		Name: "voucher-nas", Identifier: "voucher-nas", Ipaddr: "127.0.0.1",
		Secret: coaTestSecret, CoaPort: nas.port(t), Status: common.ENABLED,
	}
	require.NoError(t, db.Create(nasRow).Error)
	user := &domain.RadiusUser{
		ID: common.UUIDint64(), Username: "HOT-REVOKE", Password: "HOT-REVOKE",
		Status: common.ENABLED, ExpireTime: time.Now().Add(time.Hour),
	}
	require.NoError(t, db.Create(user).Error)
	voucher := &domain.HotspotVoucher{
		BatchID: 1, PackageID: 1, RadiusUserID: user.ID, Code: user.Username,
		Password: user.Password, ValiditySeconds: 3600, Status: "used",
	}
	require.NoError(t, db.Create(voucher).Error)
	session := &domain.RadiusOnline{
		Username: user.Username, NasId: nasRow.Identifier, NasAddr: nasRow.Ipaddr,
		AcctSessionId: "revoke-live-session", AcctStartTime: time.Now().Add(-time.Minute),
	}
	require.NoError(t, db.Create(session).Error)

	c, rec := newVoucherRevokeCtx(t, db, appCtx, voucher.ID)
	require.NoError(t, revokeVoucher(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var body voucherRevokeBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "revoked", body.Data.Status)
	assert.False(t, body.Data.AlreadyRevoked)
	assert.Equal(t, 1, body.Data.DisconnectedSessions)
	assert.Equal(t, 0, body.Data.FailedDisconnects)

	var storedVoucher domain.HotspotVoucher
	require.NoError(t, db.First(&storedVoucher, voucher.ID).Error)
	assert.Equal(t, "revoked", storedVoucher.Status)
	var storedUser domain.RadiusUser
	require.NoError(t, db.First(&storedUser, user.ID).Error)
	assert.Equal(t, common.DISABLED, storedUser.Status)

	received := nas.snapshot()
	require.Len(t, received, 1, "the live session must receive one Disconnect-Request")
	assert.Equal(t, radius.CodeDisconnectRequest, received[0].code)
	assert.Equal(t, user.Username, received[0].username)
	assert.Equal(t, session.AcctSessionId, received[0].acctSessionID)

	// A second revoke (operator retry or double click) is idempotent and must
	// not send another Disconnect-Request.
	c2, rec2 := newVoucherRevokeCtx(t, db, appCtx, voucher.ID)
	require.NoError(t, revokeVoucher(c2))
	require.Equal(t, http.StatusOK, rec2.Code)
	var body2 voucherRevokeBody
	require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &body2))
	assert.True(t, body2.Data.AlreadyRevoked)
	assert.Equal(t, 0, body2.Data.DisconnectedSessions)
	assert.Len(t, nas.snapshot(), 1)
}

func TestRevokeVoucherRejectsExpiredVoucher(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.HotspotBatch{}, &domain.HotspotVoucher{}))
	appCtx := setupTestApp(t, db)

	user := &domain.RadiusUser{
		ID: common.UUIDint64(), Username: "HOT-EXPIRED", Password: "HOT-EXPIRED",
		Status: common.ENABLED, ExpireTime: time.Now().Add(-time.Hour),
	}
	require.NoError(t, db.Create(user).Error)
	past := time.Now().Add(-time.Hour)
	voucher := &domain.HotspotVoucher{
		BatchID: 1, PackageID: 1, RadiusUserID: user.ID, Code: user.Username,
		Password: user.Password, ValiditySeconds: 3600, Status: "expired",
		FirstLoginAt: &past, ExpiresAt: &past,
	}
	require.NoError(t, db.Create(voucher).Error)

	c, rec := newVoucherRevokeCtx(t, db, appCtx, voucher.ID)
	require.NoError(t, revokeVoucher(c))
	require.Equal(t, http.StatusConflict, rec.Code)
	assert.Equal(t, "VOUCHER_EXPIRED", decodeErr(t, rec).Error)

	var storedVoucher domain.HotspotVoucher
	require.NoError(t, db.First(&storedVoucher, voucher.ID).Error)
	assert.Equal(t, "expired", storedVoucher.Status)
	var storedUser domain.RadiusUser
	require.NoError(t, db.First(&storedUser, user.ID).Error)
	assert.Equal(t, common.ENABLED, storedUser.Status)
}
