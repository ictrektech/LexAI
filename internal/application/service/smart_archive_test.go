package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"strings"
	"testing"
	"time"

	archiveRepo "github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newReminderCandidateService(t *testing.T) (*smartArchiveService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.ArchiveSettings{},
		&types.ArchiveDocument{},
		&types.ArchiveDocumentLink{},
		&types.ArchiveFieldEvidence{},
		&types.ArchiveReminder{},
		&types.ArchiveReminderOccurrence{},
		&types.ArchiveNotification{},
		&types.ArchiveReminderCandidate{},
	))
	return &smartArchiveService{repo: archiveRepo.NewSmartArchiveRepository(db)}, db
}

func reminderEvidence(documentID, field, value, quote string, confidence float64) *types.ArchiveFieldEvidence {
	return &types.ArchiveFieldEvidence{
		ID: uuid.NewString(), DocumentID: documentID, TenantID: 7, FieldName: field,
		Value: value, Quote: quote, Confidence: confidence,
		LocatorKind: types.ArchiveLocatorPDFPage, Locator: types.JSON(`{"page":1}`),
	}
}

func TestDetectReminderCandidatesDoesNotCreateFormalReminders(t *testing.T) {
	svc, db := newReminderCandidateService(t)
	doc := &types.ArchiveDocument{TenantID: 7, Title: "劳动合同", FileName: "labor.pdf", FileType: ".pdf", FileHash: "labor-hash", FilePath: "local://labor.pdf", DocumentType: types.ArchiveDocumentContract, CreatedBy: "importer"}
	require.NoError(t, db.Create(doc).Error)
	evidence := reminderEvidence(doc.ID, "expires_at", "2027年2月6日", "合同到期日：2027年2月6日", 0.95)
	require.NoError(t, svc.repo.ReplaceEvidence(context.Background(), doc.TenantID, doc.ID, []*types.ArchiveFieldEvidence{evidence}))
	fields := map[string]string{"expires_at": evidence.Value}
	require.NoError(t, svc.detectReminderCandidates(context.Background(), doc, fields, []*types.ArchiveFieldEvidence{evidence}, doc.CreatedBy))
	require.NoError(t, svc.detectReminderCandidates(context.Background(), doc, fields, []*types.ArchiveFieldEvidence{evidence}, doc.CreatedBy))

	candidates, err := svc.repo.ListReminderCandidates(context.Background(), doc.TenantID, string(types.ArchiveReminderCandidatePending))
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, types.ArchiveReminderExpiry, candidates[0].Type)
	require.Equal(t, 30, candidates[0].SuggestedOffsetDays)
	require.False(t, candidates[0].NeedsReview)
	var reminderCount int64
	require.NoError(t, db.Model(&types.ArchiveReminder{}).Where("tenant_id = ?", doc.TenantID).Count(&reminderCount).Error)
	require.Zero(t, reminderCount)
}

func TestBatchDocumentActionReportsPartialFailures(t *testing.T) {
	svc, db := newReminderCandidateService(t)
	doc := &types.ArchiveDocument{TenantID: 7, Title: "批量归档合同", FileName: "bulk.pdf", FileType: ".pdf", FileHash: "bulk-hash", FilePath: "local://bulk.pdf", ExtractionStatus: types.ArchiveExtractionCompleted, CreatedBy: "operator"}
	require.NoError(t, db.Create(doc).Error)
	result, err := svc.BatchDocumentAction(context.Background(), 7, []string{doc.ID, "missing-document"}, types.ArchiveBulkArchive)
	require.NoError(t, err)
	require.Equal(t, 2, result.Requested)
	require.Equal(t, 1, result.Succeeded)
	require.Equal(t, 1, result.Failed)
	var stored types.ArchiveDocument
	require.NoError(t, db.First(&stored, "id = ?", doc.ID).Error)
	require.NotNil(t, stored.ArchivedAt)
	failed := 0
	for _, item := range result.Items {
		if !item.Success {
			failed++
			require.NotEmpty(t, item.Error)
		}
	}
	require.Equal(t, 1, failed)
}

func TestBatchPurgePermanentlyDeletesArchivedDocumentsOnly(t *testing.T) {
	svc, db := newReminderCandidateService(t)
	archivedAt := time.Now().UTC()
	archived := &types.ArchiveDocument{TenantID: 7, Title: "已归档合同", FileName: "archived.pdf", FileType: ".pdf", FileHash: "archived-hash", ExtractionStatus: types.ArchiveExtractionCompleted, ArchivedAt: &archivedAt, CreatedBy: "operator"}
	active := &types.ArchiveDocument{TenantID: 7, Title: "活动合同", FileName: "active.pdf", FileType: ".pdf", FileHash: "active-hash", ExtractionStatus: types.ArchiveExtractionCompleted, CreatedBy: "operator"}
	require.NoError(t, db.Create(archived).Error)
	require.NoError(t, db.Create(active).Error)

	result, err := svc.BatchDocumentAction(context.Background(), 7, []string{archived.ID, active.ID}, types.ArchiveBulkPurge)
	require.NoError(t, err)
	require.Equal(t, 2, result.Requested)
	require.Equal(t, 1, result.Succeeded)
	require.Equal(t, 1, result.Failed)

	var deleted types.ArchiveDocument
	require.ErrorIs(t, db.First(&deleted, "id = ?", archived.ID).Error, gorm.ErrRecordNotFound)
	var retained types.ArchiveDocument
	require.NoError(t, db.First(&retained, "id = ?", active.ID).Error)
}

func TestDeleteDocumentCancelsReminderAndClearsNotifications(t *testing.T) {
	svc, db := newReminderCandidateService(t)
	doc := &types.ArchiveDocument{TenantID: 7, Title: "交付合同", FileName: "delivery.pdf", FileType: ".pdf", FileHash: "delivery-hash", ExtractionStatus: types.ArchiveExtractionCompleted, CreatedBy: "operator"}
	reminder := &types.ArchiveReminder{TenantID: 7, DocumentID: doc.ID, AssigneeID: "user-1", Type: types.ArchiveReminderDelivery, Title: "交付节点提醒", Rule: types.JSON(`{}`), Status: types.ArchiveReminderActive, CreatedBy: "operator"}
	candidate := &types.ArchiveReminderCandidate{TenantID: 7, DocumentID: doc.ID, Type: types.ArchiveReminderDelivery, SourceField: "delivery_at", EventAt: time.Now().UTC(), Title: "交付节点提醒", Rule: types.JSON(`{}`), Locator: types.JSON(`{}`), Fingerprint: "delivery-candidate", CreatedBy: "operator"}
	require.NoError(t, db.Create(doc).Error)
	reminder.DocumentID = doc.ID
	require.NoError(t, db.Create(reminder).Error)
	require.NoError(t, db.Create(&types.ArchiveReminderOccurrence{TenantID: 7, ReminderID: reminder.ID, Fingerprint: "delivery-document-occurrence", DueAt: time.Now().UTC()}).Error)
	require.NoError(t, db.Create(&types.ArchiveNotification{TenantID: 7, UserID: "user-1", ReminderID: reminder.ID, Title: reminder.Title}).Error)
	candidate.DocumentID = doc.ID
	require.NoError(t, db.Create(candidate).Error)

	require.NoError(t, svc.DeleteDocument(context.Background(), 7, doc.ID))

	var storedDoc types.ArchiveDocument
	require.NoError(t, db.First(&storedDoc, "id = ?", doc.ID).Error)
	require.NotNil(t, storedDoc.TrashedAt)
	var storedReminder types.ArchiveReminder
	require.NoError(t, db.First(&storedReminder, "id = ?", reminder.ID).Error)
	require.Equal(t, types.ArchiveReminderCanceled, storedReminder.Status)
	var notificationCount int64
	require.NoError(t, db.Model(&types.ArchiveNotification{}).Where("reminder_id = ?", reminder.ID).Count(&notificationCount).Error)
	require.Zero(t, notificationCount)
	var occurrenceCount int64
	require.NoError(t, db.Model(&types.ArchiveReminderOccurrence{}).Where("reminder_id = ?", reminder.ID).Count(&occurrenceCount).Error)
	require.Zero(t, occurrenceCount)
	var storedCandidate types.ArchiveReminderCandidate
	require.NoError(t, db.First(&storedCandidate, "id = ?", candidate.ID).Error)
	require.Equal(t, types.ArchiveReminderCandidateSuperseded, storedCandidate.Status)
}

func TestDeleteDocumentSucceedsWhenReminderCandidateCleanupIsUnavailable(t *testing.T) {
	svc, db := newReminderCandidateService(t)
	doc := &types.ArchiveDocument{
		TenantID: 7, Title: "待清理合同", FileName: "pending-cleanup.pdf", FileType: ".pdf",
		FileHash: "pending-cleanup-hash", ExtractionStatus: types.ArchiveExtractionCompleted,
		CreatedBy: "operator",
	}
	require.NoError(t, db.Create(doc).Error)
	// Simulate an instance that has started serving the new delete behavior
	// before the reminder-candidate migration has been applied. The document
	// must still be moved to trash; the cleanup will be retried later.
	require.NoError(t, db.Migrator().DropTable(&types.ArchiveReminderCandidate{}))

	require.NoError(t, svc.DeleteDocument(context.Background(), 7, doc.ID))

	var stored types.ArchiveDocument
	require.NoError(t, db.First(&stored, "id = ?", doc.ID).Error)
	require.NotNil(t, stored.TrashedAt)
}

func TestBatchDeleteRemindersReportsPartialFailuresAndSignalsScheduler(t *testing.T) {
	svc, db := newReminderCandidateService(t)
	due := time.Now().UTC().Add(time.Hour)
	active := &types.ArchiveReminder{TenantID: 7, AssigneeID: "user-1", Type: types.ArchiveReminderExpiry, Title: "active", Rule: types.JSON(`{}`), Status: types.ArchiveReminderActive, DueAt: &due, CreatedBy: "user-1"}
	draft := &types.ArchiveReminder{TenantID: 7, AssigneeID: "user-1", Type: types.ArchiveReminderExpiry, Title: "draft", Rule: types.JSON(`{}`), Status: types.ArchiveReminderDraft, CreatedBy: "user-1"}
	require.NoError(t, db.Create(active).Error)
	require.NoError(t, db.Create(draft).Error)
	result, err := svc.BatchDeleteReminders(context.Background(), 7, []string{active.ID, active.ID, draft.ID, "missing"})
	require.NoError(t, err)
	require.Equal(t, 3, result.Requested)
	require.Equal(t, 2, result.Succeeded)
	require.Equal(t, 1, result.Failed)
	_, ok := <-svc.ReminderWakeups()
	require.True(t, ok)
	var count int64
	require.NoError(t, db.Model(&types.ArchiveReminder{}).Where("tenant_id = ?", 7).Count(&count).Error)
	require.Zero(t, count)
	var otherTenant types.ArchiveReminder
	require.NoError(t, db.Create(&types.ArchiveReminder{TenantID: 8, AssigneeID: "user-2", Type: types.ArchiveReminderExpiry, Title: "other", Rule: types.JSON(`{}`), Status: types.ArchiveReminderActive, CreatedBy: "user-2"}).Error)
	require.NoError(t, db.First(&otherTenant, "tenant_id = ?", 8).Error)
	result, err = svc.BatchDeleteReminders(context.Background(), 7, []string{otherTenant.ID})
	require.NoError(t, err)
	require.Equal(t, 0, result.Succeeded)
	require.Equal(t, 1, result.Failed)
}

func TestBatchIgnoreReminderCandidatesOnlyIgnoresPending(t *testing.T) {
	svc, db := newReminderCandidateService(t)
	newCandidate := func(status types.ArchiveReminderCandidateStatus) *types.ArchiveReminderCandidate {
		row := &types.ArchiveReminderCandidate{TenantID: 7, DocumentID: uuid.NewString(), Type: types.ArchiveReminderExpiry, SourceField: "expires_at", EventAt: time.Now().UTC(), Title: "candidate", Confidence: .95, Locator: types.JSON(`{}`), Rule: types.JSON(`{}`), Fingerprint: uuid.NewString(), CreatedBy: "user-1", Status: status}
		require.NoError(t, db.Create(row).Error)
		return row
	}
	pending := newCandidate(types.ArchiveReminderCandidatePending)
	created := newCandidate(types.ArchiveReminderCandidateCreated)
	result, err := svc.BatchIgnoreReminderCandidates(context.Background(), 7, []string{pending.ID, created.ID, "missing"})
	require.NoError(t, err)
	require.Equal(t, 3, result.Requested)
	require.Equal(t, 1, result.Succeeded)
	require.Equal(t, 2, result.Failed)
	var stored types.ArchiveReminderCandidate
	require.NoError(t, db.First(&stored, "id = ?", pending.ID).Error)
	require.Equal(t, types.ArchiveReminderCandidateIgnored, stored.Status)
	stored = types.ArchiveReminderCandidate{}
	require.NoError(t, db.First(&stored, "id = ?", created.ID).Error)
	require.Equal(t, types.ArchiveReminderCandidateCreated, stored.Status)
	rows, err := svc.ListReminderCandidates(context.Background(), 7, string(types.ArchiveReminderCandidateIgnored))
	require.NoError(t, err)
	require.Len(t, rows, 1)
}

func TestDetectReminderCandidatesCreatesLoanReturnAndConditionSuggestions(t *testing.T) {
	svc, db := newReminderCandidateService(t)
	doc := &types.ArchiveDocument{TenantID: 7, Title: "设备借用协议", FileName: "loan.pdf", FileType: ".pdf", FileHash: "loan-hash", FilePath: "local://loan.pdf", DocumentType: types.ArchiveDocumentLoanAgreement, CreatedBy: "importer"}
	require.NoError(t, db.Create(doc).Error)
	evidence := reminderEvidence(doc.ID, "return_due_at", "2027年2月6日", "应归还日期：2027年2月6日", 0.92)
	evidenceRows := []*types.ArchiveFieldEvidence{evidence}
	require.NoError(t, svc.repo.ReplaceEvidence(context.Background(), doc.TenantID, doc.ID, evidenceRows))
	fields := map[string]string{"return_due_at": evidence.Value}
	require.NoError(t, svc.detectReminderCandidates(context.Background(), doc, fields, evidenceRows, doc.CreatedBy))

	candidates, err := svc.repo.ListReminderCandidates(context.Background(), doc.TenantID, string(types.ArchiveReminderCandidatePending))
	require.NoError(t, err)
	require.Len(t, candidates, 2)
	seen := map[types.ArchiveReminderType]bool{}
	for _, candidate := range candidates {
		seen[candidate.Type] = true
	}
	require.True(t, seen[types.ArchiveReminderReturn])
	require.True(t, seen[types.ArchiveReminderMissingReturn])

	// An actual return suppresses both future return suggestions without
	// deleting any already-created formal reminder history.
	fields["returned_at"] = "2027年2月1日"
	returnedEvidence := reminderEvidence(doc.ID, "returned_at", fields["returned_at"], "实际归还日期：2027年2月1日", 0.95)
	evidenceRows = append(evidenceRows, returnedEvidence)
	require.NoError(t, svc.detectReminderCandidates(context.Background(), doc, fields, evidenceRows, doc.CreatedBy))
	candidates, err = svc.repo.ListReminderCandidates(context.Background(), doc.TenantID, string(types.ArchiveReminderCandidatePending))
	require.NoError(t, err)
	require.Empty(t, candidates)
	var superseded int64
	require.NoError(t, db.Model(&types.ArchiveReminderCandidate{}).Where("status = ?", types.ArchiveReminderCandidateSuperseded).Count(&superseded).Error)
	require.Equal(t, int64(2), superseded)
}

func TestBackfillReminderCandidatesRepairsBoldLoanDeadline(t *testing.T) {
	svc, db := newReminderCandidateService(t)
	doc := &types.ArchiveDocument{TenantID: 7, Title: "借用协议.jpg", FileName: "借用协议.jpg", FileType: ".jpg", FileHash: "loan-bold-hash", FilePath: "local://loan-bold.jpg", DocumentType: types.ArchiveDocumentLoanAgreement, BusinessType: types.ArchiveBusinessLoan, ExtractionStatus: types.ArchiveExtractionCompleted, ExtractedText: "# 借用协议\n乙方（借用方）：客户\n本产品借用期为 1 个月，自 **2025 年 8 月 1 日** 起至 **2025 年 11 月 1 日** 止。", ExtractedFields: types.JSON(`{"customer":"客户"}`), CreatedBy: "importer"}
	require.NoError(t, db.Create(doc).Error)
	require.NoError(t, svc.BackfillReminderCandidates(context.Background()))
	candidates, err := svc.repo.ListReminderCandidates(context.Background(), doc.TenantID, string(types.ArchiveReminderCandidatePending))
	require.NoError(t, err)
	require.Len(t, candidates, 2)
	for _, candidate := range candidates {
		require.Equal(t, "2025-11-01", candidate.EventAt.Format("2006-01-02"))
	}
	var stored types.ArchiveDocument
	require.NoError(t, db.First(&stored, "id = ?", doc.ID).Error)
	var fields map[string]string
	require.NoError(t, json.Unmarshal(stored.ExtractedFields, &fields))
	require.Equal(t, "2025 年 11 月 1 日", fields["return_due_at"])
	require.NotContains(t, stored.ExtractedText, "**")
	evidence, err := svc.repo.ListEvidence(context.Background(), doc.TenantID, doc.ID)
	require.NoError(t, err)
	for _, row := range evidence {
		require.NotContains(t, row.Value, "**")
		require.NotContains(t, row.Quote, "**")
	}
}

func TestCreateReminderFromCandidateStoresWorkspaceLocalTimeAsUTC(t *testing.T) {
	svc, db := newReminderCandidateService(t)
	doc := &types.ArchiveDocument{TenantID: 7, Title: "服务合同", FileName: "service.pdf", FileType: ".pdf", FileHash: "service-hash", FilePath: "local://service.pdf", CreatedBy: "importer"}
	require.NoError(t, db.Create(doc).Error)
	candidate := &types.ArchiveReminderCandidate{TenantID: 7, DocumentID: doc.ID, Type: types.ArchiveReminderExpiry, SourceField: "expires_at", EventAt: time.Date(2027, 2, 6, 0, 0, 0, 0, time.UTC), Title: "合同到期提醒", Confidence: .95, Quote: "到期日：2027年2月6日", Locator: types.JSON(`{"page":1}`), Rule: types.JSON(`{"field":"expires_at"}`), Fingerprint: "candidate-time-fingerprint", CreatedBy: "importer"}
	require.NoError(t, svc.repo.UpsertReminderCandidate(context.Background(), candidate))
	require.NoError(t, svc.repo.SaveSettings(context.Background(), &types.ArchiveSettings{TenantID: 7, Timezone: "Asia/Shanghai"}))
	reminder, err := svc.CreateReminderFromCandidate(context.Background(), 7, "importer", candidate.ID, 30, "09:00", "importer")
	require.NoError(t, err)
	require.Equal(t, types.ArchiveReminderDraft, reminder.Status)
	require.Equal(t, "2027-01-07T01:00:00Z", reminder.DueAt.UTC().Format(time.RFC3339))
	var rule map[string]any
	require.NoError(t, json.Unmarshal(reminder.Rule, &rule))
	require.Equal(t, float64(30), rule["offset_days"])
}

func TestParseArchiveDateSupportsChineseFormats(t *testing.T) {
	for _, input := range []string{"2024年02月06日", "2024年2月6日", "2024-02-06"} {
		got, err := parseArchiveDate(input)
		if err != nil {
			t.Fatalf("parseArchiveDate(%q): %v", input, err)
		}
		if got.Format("2006-01-02") != "2024-02-06" {
			t.Fatalf("parseArchiveDate(%q) = %s", input, got.Format("2006-01-02"))
		}
	}
	if _, err := parseArchiveDate("2024年2月31日"); err == nil {
		t.Fatal("invalid calendar date should be rejected")
	}
}

func TestParseArchiveFieldDateUsesDeadlineEndDate(t *testing.T) {
	got, err := parseArchiveFieldDate("expires_at", "2024年2月6日至2025年2月6日")
	if err != nil {
		t.Fatalf("parseArchiveFieldDate: %v", err)
	}
	if got.Format("2006-01-02") != "2025-02-06" {
		t.Fatalf("deadline = %s", got.Format("2006-01-02"))
	}
}

func TestExtractArchiveFieldsKeepsReturnDueSeparateFromActualReturn(t *testing.T) {
	doc := &types.ArchiveDocument{FileName: "设备借用协议.pdf", FileType: ".pdf"}
	fields, evidence := extractArchiveFields(doc, "设备借用协议\n应归还日期：2025年2月6日\n实际归还日期：2025年2月10日")
	if fields["return_due_at"] != "2025年2月6日" {
		t.Fatalf("return_due_at = %q", fields["return_due_at"])
	}
	if fields["returned_at"] != "2025年2月10日" {
		t.Fatalf("returned_at = %q", fields["returned_at"])
	}
	if doc.ReturnDueAt == nil || doc.ReturnDueAt.Format("2006-01-02") != "2025-02-06" {
		t.Fatalf("return_due_at = %v", doc.ReturnDueAt)
	}
	if len(evidence) < 2 {
		t.Fatalf("evidence count = %d", len(evidence))
	}
}

func TestExtractArchiveFieldsDoesNotTreatExpectedReturnAsActualReturn(t *testing.T) {
	doc := &types.ArchiveDocument{FileName: "设备借用协议.pdf", FileType: ".pdf"}
	fields, evidence := extractArchiveFields(doc, "设备借用协议\n归还日期：2025年2月6日")
	if fields["return_due_at"] != "2025年2月6日" {
		t.Fatalf("return_due_at = %q", fields["return_due_at"])
	}
	if fields["returned_at"] != "" {
		t.Fatalf("returned_at = %q", fields["returned_at"])
	}
	if len(evidence) != 1 || evidence[0].FieldName != "return_due_at" {
		t.Fatalf("unexpected evidence: %#v", evidence)
	}
}

func TestExtractLoanAgreementDateRangeAndBorrowerCreateReturnSource(t *testing.T) {
	doc := &types.ArchiveDocument{TenantID: 7, ID: uuid.NewString(), FileName: "借用协议.jpg", FileType: ".jpg"}
	content := "借用协议\n甲方（出借方）：芯途异构科技（深圳）有限公司\n乙方（借用方）：广东利通科技投资有限公司智慧交通研发中心\n本产品借用期为1个月，自 **2025年8月1日** 起至 **2025年11月1日** 止；乙方承诺在借用期限截止日期之前将设备归还甲方。\n| 序号 | 物料名称 | 物料参数 | 数量 | 备注 |\n| --- | --- | --- | --- | --- |\n| 1 | AI计算盒 | AI计算盒 SE50221（ES33000180） | 1 | - |"
	fields, evidence := extractArchiveFields(doc, content)
	require.Equal(t, types.ArchiveDocumentLoanAgreement, doc.DocumentType)
	require.Equal(t, "广东利通科技投资有限公司智慧交通研发中心", fields["borrower"])
	require.Equal(t, fields["borrower"], fields["customer"])
	require.Equal(t, "2025年11月1日", fields["return_due_at"])
	require.Equal(t, "AI计算盒", fields["asset_name"])
	require.Equal(t, "AI计算盒 SE50221（ES33000180）", fields["asset_model"])
	require.Equal(t, "ES33000180", fields["serial_number"])
	require.Equal(t, "1", fields["quantity"])
	require.NotNil(t, doc.ReturnDueAt)
	require.Equal(t, "2025-11-01", doc.ReturnDueAt.Format("2006-01-02"))
	fieldsByName := map[string]bool{}
	for _, row := range evidence {
		fieldsByName[row.FieldName] = true
		require.Equal(t, types.ArchiveLocatorImage, row.LocatorKind)
	}
	require.True(t, fieldsByName["return_due_at"])
	require.True(t, fieldsByName["customer"])
	require.True(t, fieldsByName["serial_number"])
}

func TestPrepareArchiveOCRImageReducesCameraOriginalOnlyForOCR(t *testing.T) {
	// A deliberately large solid JPEG is easy to encode and exercises the
	// dimension/size guard without requiring an external image fixture.
	src := image.NewRGBA(image.Rect(0, 0, 5712, 4284))
	var original bytes.Buffer
	require.NoError(t, jpeg.Encode(&original, src, &jpeg.Options{Quality: 100}))
	prepared := prepareArchiveOCRImage(storedArchiveUpload{name: "loan.jpg", data: original.Bytes()})
	require.NotEmpty(t, prepared)
	require.Less(t, len(prepared), len(original.Bytes()))
	config, _, err := image.DecodeConfig(bytes.NewReader(prepared))
	require.NoError(t, err)
	require.LessOrEqual(t, config.Width, archiveOCRMaxDimension)
	require.LessOrEqual(t, config.Height, archiveOCRMaxDimension)
	oriented := orientArchiveOCRImage(image.NewRGBA(image.Rect(0, 0, 2, 3)), 6)
	orientedBounds := oriented.Bounds()
	require.Equal(t, 3, orientedBounds.Dx())
	require.Equal(t, 2, orientedBounds.Dy())
}

func TestReturnedLoanDoesNotProduceReturnReminderDate(t *testing.T) {
	doc := &types.ArchiveDocument{FileName: "设备借用协议.pdf", FileType: ".pdf"}
	fields, evidence := extractArchiveFields(doc, "设备借用协议\n设备归还日期：2025年2月6日\n实际归还日期：2025年2月10日")
	if fields["return_due_at"] != "2025年2月6日" || fields["returned_at"] != "2025年2月10日" {
		t.Fatalf("unexpected return fields: %#v", fields)
	}
	// Candidate generation is repository-backed and covered by the service
	// integration path; this assertion documents the parser invariant that
	// lets it suppress already-returned loans.
	if len(evidence) < 2 {
		t.Fatalf("evidence count = %d", len(evidence))
	}
}

func TestClassifyArchiveDocumentPrefersDocumentTitleOverClauseTerms(t *testing.T) {
	tests := []struct {
		name         string
		fileName     string
		content      string
		documentType types.ArchiveDocumentType
		businessType types.ArchiveBusinessType
	}{
		{
			name:         "contract with payment clause",
			fileName:     "呼和浩特市文物保护中心-政府采购服务合同.pdf",
			content:      "政府采购合同\n合同编号：TZCHH2019-GK-0446-01\n九、付款方式：每季度结算一次。",
			documentType: types.ArchiveDocumentContract,
			businessType: types.ArchiveBusinessSale,
		},
		{
			name:         "payment voucher",
			fileName:     "2024-01-01-付款凭证.pdf",
			content:      "付款凭证\n合同编号：A-100\n付款金额：1000",
			documentType: types.ArchiveDocumentPayment,
		},
		{
			name:         "loan agreement with return clause",
			fileName:     "设备借用协议.docx",
			content:      "设备借用协议\n设备归还日期：2025-01-01",
			documentType: types.ArchiveDocumentLoanAgreement,
			businessType: types.ArchiveBusinessLoan,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := &types.ArchiveDocument{FileName: tt.fileName}
			classifyArchiveDocument(doc, tt.content)
			if doc.DocumentType != tt.documentType {
				t.Fatalf("document type = %q, want %q", doc.DocumentType, tt.documentType)
			}
			if tt.businessType != "" && doc.BusinessType != tt.businessType {
				t.Fatalf("business type = %q, want %q", doc.BusinessType, tt.businessType)
			}
		})
	}
}

func TestGovernmentPurchaseContractExtractsBuyerWithoutRoleSymbols(t *testing.T) {
	doc := &types.ArchiveDocument{FileName: "吉林市应急管理局-政府采购服务合同.pdf", FileType: ".pdf"}
	fields, evidence := extractArchiveFields(doc, "政府采购服务合同\n合同各方：\n甲方（买方）：吉林市应急管理局\n乙方：吉林市江城保安集团有限责任公司")
	if fields["customer"] != "吉林市应急管理局" {
		t.Fatalf("customer = %q, want clean buyer name", fields["customer"])
	}
	if fields["borrower"] != "吉林市江城保安集团有限责任公司" {
		t.Fatalf("borrower = %q", fields["borrower"])
	}
	for _, row := range evidence {
		if row.FieldName == "customer" && strings.ContainsAny(row.Value, "：:*`") {
			t.Fatalf("customer evidence contains presentation symbols: %q", row.Value)
		}
	}
}

func TestCleanArchiveEntityNameRejectsRoleLabels(t *testing.T) {
	tests := map[string]string{
		"乙方：":        "",
		"**甲方：乙方：**": "",
		"• 采购人：吉林市应急管理局 |": "吉林市应急管理局",
		"供应商：某某科技有限公司":     "某某科技有限公司",
		"客户服务有限公司":         "客户服务有限公司",
		"供应商贸有限公司":         "供应商贸有限公司",
	}
	for input, want := range tests {
		if got := cleanArchiveEntityName(input); got != want {
			t.Fatalf("cleanArchiveEntityName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestArchiveImageExtensionsAndContentValidation(t *testing.T) {
	for _, ext := range []string{".jpg", ".jpeg", ".png", ".webp"} {
		if !archiveExtensionAllowed(ext) || !archiveImageExtension(ext) || archiveImageMIME(ext) == "" {
			t.Fatalf("image extension %q was not registered", ext)
		}
	}
	for _, ext := range []string{".gif", ".bmp", ".tiff"} {
		if archiveExtensionAllowed(ext) || archiveImageExtension(ext) {
			t.Fatalf("unsupported image extension %q was accepted", ext)
		}
	}
	// A 1x1 transparent PNG is enough to exercise signature validation without
	// depending on an image fixture on disk.
	pngData, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	require.NoError(t, err)
	require.NoError(t, validateArchiveUploadContent(".png", pngData))
	require.NoError(t, validateArchiveUploadMIME(".jpg", "image/jpeg"))
	require.NoError(t, validateArchiveUploadMIME(".jpg", "image/jpg"))
	require.ErrorIs(t, validateArchiveUploadMIME(".jpg", "image/png"), ErrArchiveInvalidFile)
	require.ErrorIs(t, validateArchiveUploadContent(".jpg", pngData), ErrArchiveInvalidFile)
	require.ErrorIs(t, validateArchiveUploadContent(".webp", pngData), ErrArchiveInvalidFile)
	require.ErrorIs(t, validateArchiveUploadContent(".png", []byte("not an image")), ErrArchiveInvalidFile)
}

func TestParseArchiveImageUsesOCRAndReviewErrorWhenUnavailable(t *testing.T) {
	svc := &smartArchiveService{imageOCR: func(_ context.Context, upload storedArchiveUpload) (string, error) {
		require.Equal(t, "contract.png", upload.name)
		return "合同编号：IMG-001\n合同到期日：2027年2月6日", nil
	}}
	text, err := svc.parse(context.Background(), storedArchiveUpload{name: "contract.png", data: []byte("image")})
	require.NoError(t, err)
	require.Contains(t, text, "IMG-001")

	withoutModel := &smartArchiveService{}
	_, err = withoutModel.parse(context.Background(), storedArchiveUpload{name: "contract.png", data: []byte("image")})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrArchiveImageOCRNeedsReview)
}

func TestExtractArchiveFieldsUsesImageEvidenceLocator(t *testing.T) {
	doc := &types.ArchiveDocument{TenantID: 7, FileName: "contract.png", FileType: ".png"}
	fields, evidence := extractArchiveFields(doc, "合同编号：IMG-001\n合同到期日：2027年2月6日")
	require.Equal(t, "IMG-001", fields["agreement_number"])
	require.NotEmpty(t, evidence)
	for _, row := range evidence {
		require.Equal(t, types.ArchiveLocatorImage, row.LocatorKind)
		var locator map[string]any
		require.NoError(t, json.Unmarshal(row.Locator, &locator))
		require.Equal(t, float64(1), locator["page"])
	}
}
