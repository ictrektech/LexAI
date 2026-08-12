package interfaces

import (
	"context"
	"io"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

type ArchiveRepository interface {
	GetSettings(context.Context, uint64) (*types.ArchiveSettings, error)
	SaveSettings(context.Context, *types.ArchiveSettings) error
	CreateBatch(context.Context, *types.ArchiveImportBatch) error
	GetBatch(context.Context, uint64, string) (*types.ArchiveImportBatch, error)
	UpdateBatch(context.Context, *types.ArchiveImportBatch) error
	CreateDocument(context.Context, *types.ArchiveDocument) error
	GetDocument(context.Context, uint64, string) (*types.ArchiveDocument, error)
	ListDocuments(context.Context, uint64, string, bool) ([]*types.ArchiveDocument, error)
	UpdateDocument(context.Context, *types.ArchiveDocument) error
	DeleteDocument(context.Context, uint64, string) error
	FindDocumentByHash(context.Context, uint64, string) (*types.ArchiveDocument, error)
	CreateCustomer(context.Context, *types.ArchiveCustomer) error
	FindCustomer(context.Context, uint64, string) (*types.ArchiveCustomer, error)
	ListCustomers(context.Context, uint64, string) ([]*types.ArchiveCustomer, error)
	UpdateCustomer(context.Context, *types.ArchiveCustomer) error
	CreateAsset(context.Context, *types.ArchiveAsset) error
	FindAssetBySN(context.Context, uint64, string) (*types.ArchiveAsset, error)
	ListAssets(context.Context, uint64, string) ([]*types.ArchiveAsset, error)
	UpdateAsset(context.Context, *types.ArchiveAsset) error
	LinkDocumentAsset(context.Context, *types.ArchiveDocumentAsset) error
	CreateDocumentLink(context.Context, *types.ArchiveDocumentLink) error
	ListDocumentLinks(context.Context, uint64, string) ([]*types.ArchiveDocumentLink, error)
	ListDocumentAssets(context.Context, uint64, string) ([]*types.ArchiveAsset, error)
	ReplaceEvidence(context.Context, uint64, string, []*types.ArchiveFieldEvidence) error
	ListEvidence(context.Context, uint64, string) ([]*types.ArchiveFieldEvidence, error)
	CreateReminder(context.Context, *types.ArchiveReminder) error
	GetReminder(context.Context, uint64, string) (*types.ArchiveReminder, error)
	ListReminders(context.Context, uint64, string) ([]*types.ArchiveReminder, error)
	UpdateReminder(context.Context, *types.ArchiveReminder) error
	DeleteReminder(context.Context, uint64, string) error
	ListDueReminders(context.Context, int) ([]*types.ArchiveReminder, error)
	CreateOccurrence(context.Context, *types.ArchiveReminderOccurrence) (bool, error)
	NextReminderWakeAt(context.Context) (*time.Time, error)
	DeliverReminder(context.Context, *types.ArchiveReminder, *types.ArchiveReminderOccurrence, *types.ArchiveNotification) error
	ListTrashedDocuments(context.Context) ([]*types.ArchiveDocument, error)
	HardDeleteDocument(context.Context, uint64, string) error
	CreateNotification(context.Context, *types.ArchiveNotification) error
	ListNotifications(context.Context, uint64, string, bool) ([]*types.ArchiveNotification, error)
	MarkNotificationRead(context.Context, uint64, string, string) error
	ListReminderCandidates(context.Context, uint64, string) ([]*types.ArchiveReminderCandidate, error)
	GetReminderCandidate(context.Context, uint64, string) (*types.ArchiveReminderCandidate, error)
	UpsertReminderCandidate(context.Context, *types.ArchiveReminderCandidate) error
	UpdateReminderCandidate(context.Context, *types.ArchiveReminderCandidate) error
	IgnoreReminderCandidate(context.Context, uint64, string) error
	CreateReminderFromCandidate(context.Context, *types.ArchiveReminderCandidate, *types.ArchiveReminder) error
	Search(context.Context, uint64, *types.ArchiveSearchRequest) (*types.ArchiveSearchResponse, error)
	ListCompletedDocuments(context.Context) ([]*types.ArchiveDocument, error)
}

type SmartArchiveService interface {
	GetSettings(context.Context, uint64) (*types.ArchiveSettings, error)
	UpdateSettings(context.Context, uint64, *types.ArchiveSettings) (*types.ArchiveSettings, error)
	Import(context.Context, uint64, string, []*types.ArchiveUpload) (*types.ArchiveImportBatch, error)
	GetBatch(context.Context, uint64, string) (*types.ArchiveImportBatch, error)
	GetDocument(context.Context, uint64, string) (*types.ArchiveDocument, error)
	ListDocuments(context.Context, uint64, string, bool) ([]*types.ArchiveDocument, error)
	UpdateDocument(context.Context, uint64, string, map[string]any) (*types.ArchiveDocument, error)
	RetryExtraction(context.Context, uint64, string, string) (*types.ArchiveDocument, error)
	ArchiveDocument(context.Context, uint64, string, bool) (*types.ArchiveDocument, error)
	DeleteDocument(context.Context, uint64, string) error
	BatchDocumentAction(context.Context, uint64, []string, types.ArchiveBulkAction) (*types.ArchiveBulkActionResult, error)
	OpenDocument(context.Context, uint64, string) (io.ReadCloser, string, error)
	ListCustomers(context.Context, uint64, string) ([]*types.ArchiveCustomer, error)
	UpdateCustomer(context.Context, uint64, string, map[string]any) (*types.ArchiveCustomer, error)
	ListAssets(context.Context, uint64, string) ([]*types.ArchiveAsset, error)
	UpdateAsset(context.Context, uint64, string, map[string]any) (*types.ArchiveAsset, error)
	ListEvidence(context.Context, uint64, string) ([]*types.ArchiveFieldEvidence, error)
	Search(context.Context, uint64, *types.ArchiveSearchRequest) (*types.ArchiveSearchResponse, error)
	ListReminders(context.Context, uint64, string) ([]*types.ArchiveReminder, error)
	CreateReminder(context.Context, uint64, string, *types.ArchiveReminder) (*types.ArchiveReminder, error)
	UpdateReminderStatus(context.Context, uint64, string, types.ArchiveReminderStatus, *time.Time, string) (*types.ArchiveReminder, error)
	DeleteReminder(context.Context, uint64, string) error
	BatchDeleteReminders(context.Context, uint64, []string) (*types.ArchiveBulkActionResult, error)
	ListReminderCandidates(context.Context, uint64, string) ([]*types.ArchiveReminderCandidate, error)
	CreateReminderFromCandidate(context.Context, uint64, string, string, int, string, string) (*types.ArchiveReminder, error)
	BatchIgnoreReminderCandidates(context.Context, uint64, []string) (*types.ArchiveBulkActionResult, error)
	BackfillReminderCandidates(context.Context) error
	ListNotifications(context.Context, uint64, string, bool) ([]*types.ArchiveNotification, error)
	MarkNotificationRead(context.Context, uint64, string, string) error
	RunDueReminders(context.Context) error
	NextReminderWakeAt(context.Context) (*time.Time, error)
	ReminderWakeups() <-chan struct{}
}
