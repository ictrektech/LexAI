package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newSmartArchiveSearchRepository(t *testing.T) (interfaces.ArchiveRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.ArchiveDocument{},
		&types.ArchiveCustomer{},
		&types.ArchiveAsset{},
		&types.ArchiveDocumentAsset{},
		&types.ArchiveDocumentLink{},
		&types.ArchiveFieldEvidence{},
		&types.ArchiveReminder{},
		&types.ArchiveReminderCandidate{},
		&types.ArchiveReminderOccurrence{},
		&types.ArchiveNotification{},
	))
	// Production migrations use a partial unique index so legacy empty
	// occurrence IDs remain valid. Recreate that constraint for the in-memory
	// repository tests as well.
	require.NoError(t, db.Exec("CREATE UNIQUE INDEX idx_test_archive_notification_occurrence ON archive_notifications(occurrence_id) WHERE occurrence_id IS NOT NULL AND occurrence_id <> ''").Error)
	return NewSmartArchiveRepository(db), db
}

func TestReminderCandidateCreateIsTransactionalAndIdempotent(t *testing.T) {
	repo, db := newSmartArchiveSearchRepository(t)
	document := &types.ArchiveDocument{TenantID: 7, Title: "服务合同", FileName: "contract.pdf", FileType: ".pdf", FileHash: "candidate-hash", FilePath: "local://contract.pdf", CreatedBy: "user-1"}
	require.NoError(t, db.Create(document).Error)
	candidate := &types.ArchiveReminderCandidate{
		TenantID: 7, DocumentID: document.ID, DocumentTitle: document.Title,
		Type: types.ArchiveReminderExpiry, SourceField: "expires_at",
		EventAt: time.Date(2027, 2, 6, 0, 0, 0, 0, time.UTC), SuggestedOffsetDays: 30,
		Title: "合同到期提醒", Description: "请确认后创建", Confidence: 0.95,
		Quote: "合同到期日：2027年2月6日", Rule: types.JSON(`{"field":"expires_at"}`), CreatedBy: "user-1",
	}
	require.NoError(t, repo.UpsertReminderCandidate(context.Background(), candidate))
	rows, err := repo.ListReminderCandidates(context.Background(), 7, "pending")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, candidate.ID, rows[0].ID)

	reminder := &types.ArchiveReminder{
		TenantID: 7, DocumentID: document.ID, AssigneeID: "user-1", Type: types.ArchiveReminderExpiry,
		Title: candidate.Title, Description: candidate.Description, Rule: types.JSON(`{"offset_days":30}`), Status: types.ArchiveReminderDraft, CreatedBy: "user-1",
	}
	require.NoError(t, repo.CreateReminderFromCandidate(context.Background(), candidate, reminder))
	var saved types.ArchiveReminderCandidate
	require.NoError(t, db.First(&saved, "id = ?", candidate.ID).Error)
	require.Equal(t, types.ArchiveReminderCandidateCreated, saved.Status)
	require.Equal(t, reminder.ID, saved.ReminderID)

	duplicate := &types.ArchiveReminder{TenantID: 7, DocumentID: document.ID, AssigneeID: "user-1", Type: types.ArchiveReminderExpiry, Title: candidate.Title, Rule: types.JSON(`{}`), Status: types.ArchiveReminderDraft, CreatedBy: "user-1"}
	require.ErrorIs(t, repo.CreateReminderFromCandidate(context.Background(), candidate, duplicate), gorm.ErrDuplicatedKey)
	var reminderCount int64
	require.NoError(t, db.Model(&types.ArchiveReminder{}).Where("tenant_id = ?", 7).Count(&reminderCount).Error)
	require.Equal(t, int64(1), reminderCount)

	otherTenant, err := repo.ListReminderCandidates(context.Background(), 8, "pending")
	require.NoError(t, err)
	require.Empty(t, otherTenant)
}

func TestSmartArchiveSearchOrdersWithoutDistinctConflict(t *testing.T) {
	repo, db := newSmartArchiveSearchRepository(t)
	doc := &types.ArchiveDocument{
		TenantID:      1,
		Title:         "采购服务合同",
		FileName:      "contract.pdf",
		FileType:      ".pdf",
		FileHash:      "hash-1",
		FilePath:      "local://contract.pdf",
		CreatedBy:     "user-1",
		ExtractedText: "合同正文包含付款方式条款",
	}
	require.NoError(t, db.Create(doc).Error)

	result, err := repo.Search(context.Background(), 1, &types.ArchiveSearchRequest{Query: "合同", Page: 1, PageSize: 30})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Documents, 1)
	require.Equal(t, doc.ID, result.Documents[0].ID)
}

func TestSmartArchiveSearchAssetFilterDoesNotDuplicateDocuments(t *testing.T) {
	repo, db := newSmartArchiveSearchRepository(t)
	doc := &types.ArchiveDocument{
		TenantID:  1,
		Title:     "设备合同",
		FileName:  "asset-contract.pdf",
		FileType:  ".pdf",
		FileHash:  "hash-asset",
		FilePath:  "local://asset-contract.pdf",
		CreatedBy: "user-1",
	}
	require.NoError(t, db.Create(doc).Error)
	matching := &types.ArchiveAsset{TenantID: 1, Model: "H100", Name: "GPU"}
	other := &types.ArchiveAsset{TenantID: 1, Model: "A100", Name: "GPU"}
	require.NoError(t, db.Create(matching).Error)
	require.NoError(t, db.Create(other).Error)
	require.NoError(t, db.Create(&types.ArchiveDocumentAsset{TenantID: 1, DocumentID: doc.ID, AssetID: matching.ID}).Error)
	require.NoError(t, db.Create(&types.ArchiveDocumentAsset{TenantID: 1, DocumentID: doc.ID, AssetID: other.ID}).Error)

	result, err := repo.Search(context.Background(), 1, &types.ArchiveSearchRequest{Filters: types.ArchiveSearchFilters{Model: "H100"}, Page: 1, PageSize: 30})
	require.NoError(t, err)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, result.Documents, 1)
	require.Equal(t, doc.ID, result.Documents[0].ID)
}

func TestDeliverReminderIsTransactionalAndIdempotent(t *testing.T) {
	repo, db := newSmartArchiveSearchRepository(t)
	due := time.Now().UTC().Add(-time.Minute)
	reminder := &types.ArchiveReminder{
		TenantID: 7, AssigneeID: "user-1", Type: types.ArchiveReminderExpiry,
		Title: "合同到期提醒", Description: "请处理", Rule: types.JSON(`{"offset_days":0}`),
		Status: types.ArchiveReminderActive, DueAt: &due, CreatedBy: "user-1",
	}
	require.NoError(t, db.Create(reminder).Error)
	occurrence := &types.ArchiveReminderOccurrence{ReminderID: reminder.ID, TenantID: reminder.TenantID, Fingerprint: reminder.ID + ":" + due.Format(time.RFC3339), DueAt: due}
	notification := &types.ArchiveNotification{TenantID: reminder.TenantID, UserID: reminder.AssigneeID, ReminderID: reminder.ID, Title: reminder.Title, Body: reminder.Description}
	require.NoError(t, repo.DeliverReminder(context.Background(), reminder, occurrence, notification))
	// A second worker seeing the same due row must not create another in-app
	// notification, and it should still repair the reminder cursor.
	require.NoError(t, repo.DeliverReminder(context.Background(), reminder, occurrence, notification))
	var occurrenceCount, notificationCount int64
	require.NoError(t, db.Model(&types.ArchiveReminderOccurrence{}).Where("fingerprint = ?", occurrence.Fingerprint).Count(&occurrenceCount).Error)
	require.NoError(t, db.Model(&types.ArchiveNotification{}).Where("reminder_id = ?", reminder.ID).Count(&notificationCount).Error)
	require.Equal(t, int64(1), occurrenceCount)
	require.Equal(t, int64(1), notificationCount)
	var savedOccurrence types.ArchiveReminderOccurrence
	require.NoError(t, db.First(&savedOccurrence, "fingerprint = ?", occurrence.Fingerprint).Error)
	require.Equal(t, types.ArchiveOccurrenceSent, savedOccurrence.Status)
	var savedReminder types.ArchiveReminder
	require.NoError(t, db.First(&savedReminder, "id = ?", reminder.ID).Error)
	require.NotNil(t, savedReminder.LastOccurrenceAt)

	// A condition that is satisfied is persisted as skipped, without a
	// notification, but it still advances the cursor and will not be rescanned.
	reminder2 := &types.ArchiveReminder{TenantID: 7, AssigneeID: "user-1", Type: types.ArchiveReminderMissingReturn, Title: "检查归还", Rule: types.JSON(`{}`), Status: types.ArchiveReminderActive, DueAt: &due, CreatedBy: "user-1"}
	require.NoError(t, db.Create(reminder2).Error)
	occurrence2 := &types.ArchiveReminderOccurrence{ReminderID: reminder2.ID, TenantID: reminder2.TenantID, Fingerprint: reminder2.ID + ":" + due.Format(time.RFC3339), DueAt: due, Status: types.ArchiveOccurrenceSkipped}
	notification2 := &types.ArchiveNotification{TenantID: reminder2.TenantID, UserID: reminder2.AssigneeID, ReminderID: reminder2.ID, Title: reminder2.Title}
	require.NoError(t, repo.DeliverReminder(context.Background(), reminder2, occurrence2, notification2))
	require.NoError(t, db.Model(&types.ArchiveNotification{}).Where("reminder_id = ?", reminder2.ID).Count(&notificationCount).Error)
	require.Zero(t, notificationCount)
}

func TestNextReminderWakeAtUsesActiveDueAndSnoozeTimes(t *testing.T) {
	repo, db := newSmartArchiveSearchRepository(t)
	now := time.Now().UTC().Truncate(time.Second)
	activeDue := now.Add(2 * time.Hour)
	snoozeUntil := now.Add(time.Hour)
	active := &types.ArchiveReminder{TenantID: 7, AssigneeID: "user-1", Type: types.ArchiveReminderExpiry, Title: "active", Rule: types.JSON(`{}`), Status: types.ArchiveReminderActive, DueAt: &activeDue, CreatedBy: "user-1"}
	snoozed := &types.ArchiveReminder{TenantID: 7, AssigneeID: "user-1", Type: types.ArchiveReminderExpiry, Title: "snoozed", Rule: types.JSON(`{}`), Status: types.ArchiveReminderSnoozed, DueAt: &activeDue, SnoozedUntil: &snoozeUntil, CreatedBy: "user-1"}
	require.NoError(t, db.Create(active).Error)
	require.NoError(t, db.Create(snoozed).Error)
	wake, err := repo.NextReminderWakeAt(context.Background())
	require.NoError(t, err)
	require.NotNil(t, wake)
	require.WithinDuration(t, snoozeUntil, *wake, time.Second)
}
