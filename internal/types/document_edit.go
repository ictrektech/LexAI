package types

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DocumentEditFormat string

const (
	DocumentEditFormatDOCX DocumentEditFormat = "DOCX"
	DocumentEditFormatXLSX DocumentEditFormat = "XLSX"
	DocumentEditFormatPPTX DocumentEditFormat = "PPTX"
)

type DocumentEditMode string

const (
	DocumentEditModeAdeu      DocumentEditMode = "adeu"
	DocumentEditModeOfficeCLI DocumentEditMode = "officecli"
	DocumentEditModeHybrid    DocumentEditMode = "hybrid"
)

type DocumentEditStatus string

const (
	DocumentEditStatusQueued    DocumentEditStatus = "queued"
	DocumentEditStatusRunning   DocumentEditStatus = "running"
	DocumentEditStatusCompleted DocumentEditStatus = "completed"
	DocumentEditStatusFailed    DocumentEditStatus = "failed"
	DocumentEditStatusCancelled DocumentEditStatus = "cancelled"
)

type DocumentEditOperationStatus string

const (
	DocumentEditOperationPlanned   DocumentEditOperationStatus = "planned"
	DocumentEditOperationApplied   DocumentEditOperationStatus = "applied"
	DocumentEditOperationFailed    DocumentEditOperationStatus = "failed"
	DocumentEditOperationCancelled DocumentEditOperationStatus = "cancelled"
)

type DocumentEditComparisonStrategy string

const (
	DocumentEditComparisonReplan     DocumentEditComparisonStrategy = "replan"
	DocumentEditComparisonLockedPlan DocumentEditComparisonStrategy = "locked_plan"
)

type DocumentEditStageStatus string

const (
	DocumentEditStageRunning   DocumentEditStageStatus = "running"
	DocumentEditStageCompleted DocumentEditStageStatus = "completed"
	DocumentEditStageFailed    DocumentEditStageStatus = "failed"
	DocumentEditStageSkipped   DocumentEditStageStatus = "skipped"
)

type EditPlan struct {
	SchemaVersion string             `json:"schema_version"`
	Format        DocumentEditFormat `json:"format"`
	BaseSHA256    string             `json:"base_sha256"`
	ApplyMode     string             `json:"apply_mode"`
	OutputModes   []string           `json:"output_modes,omitempty"`
	Operations    []EditOperation    `json:"operations"`
}

type EditOperation struct {
	OperationID string      `json:"operation_id"`
	Kind        string      `json:"kind"`
	Target      EditTarget  `json:"target"`
	Payload     EditPayload `json:"payload"`
}

type EditTarget struct {
	Part            string `json:"part,omitempty"`
	AnchorID        string `json:"anchor_id,omitempty"`
	Quote           string `json:"quote"`
	Prefix          string `json:"prefix,omitempty"`
	Suffix          string `json:"suffix,omitempty"`
	ExpectedMatches int    `json:"expected_matches"`
	ParagraphSHA256 string `json:"paragraph_sha256,omitempty"`
}

type EditPayload struct {
	Text    string `json:"text,omitempty"`
	Comment string `json:"comment,omitempty"`
}

// DocumentEditJob is the durable, tenant-scoped state machine for one edit
// request. SourceRef and artifact refs are storage-provider paths, never host
// filesystem paths returned by a worker.
type DocumentEditJob struct {
	ID                 string                         `json:"id" gorm:"type:varchar(36);primaryKey"`
	TenantID           uint64                         `json:"tenant_id" gorm:"not null;index"`
	UserID             string                         `json:"user_id" gorm:"type:varchar(64);not null;index"`
	Format             DocumentEditFormat             `json:"format" gorm:"type:varchar(8);not null"`
	Mode               DocumentEditMode               `json:"mode" gorm:"type:varchar(16);not null"`
	Status             DocumentEditStatus             `json:"status" gorm:"type:varchar(16);not null;index"`
	FileName           string                         `json:"file_name" gorm:"type:varchar(1024);not null"`
	MimeType           string                         `json:"mime_type" gorm:"type:varchar(255);not null;default:''"`
	FileSize           int64                          `json:"file_size" gorm:"not null;default:0"`
	SourceSHA256       string                         `json:"source_sha256" gorm:"type:char(64);not null"`
	SourceRef          string                         `json:"-" gorm:"type:text;not null"`
	Instruction        string                         `json:"instruction" gorm:"type:text;not null"`
	ModelID            string                         `json:"model_id,omitempty" gorm:"type:varchar(64);not null;default:''"`
	Plan               JSON                           `json:"plan,omitempty" gorm:"type:jsonb"`
	Capabilities       JSON                           `json:"capabilities,omitempty" gorm:"type:jsonb"`
	ComparisonGroupID  string                         `json:"comparison_group_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	ComparisonParentID string                         `json:"comparison_parent_id,omitempty" gorm:"type:varchar(36);not null;default:'';index"`
	ComparisonStrategy DocumentEditComparisonStrategy `json:"comparison_strategy,omitempty" gorm:"type:varchar(16);not null;default:''"`
	ErrorCode          string                         `json:"error_code,omitempty" gorm:"type:varchar(64);not null;default:''"`
	ErrorMessage       string                         `json:"error_message,omitempty" gorm:"type:text;not null;default:''"`
	StartedAt          *time.Time                     `json:"started_at,omitempty"`
	CompletedAt        *time.Time                     `json:"completed_at,omitempty"`
	CreatedAt          time.Time                      `json:"created_at"`
	UpdatedAt          time.Time                      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt                 `json:"-" gorm:"index"`
	Artifacts          []*DocumentEditArtifact        `json:"artifacts,omitempty" gorm:"foreignKey:JobID"`
	Operations         []*DocumentEditOperation       `json:"operations,omitempty" gorm:"foreignKey:JobID"`
}

func (j *DocumentEditJob) BeforeCreate(_ *gorm.DB) error {
	if j.ID == "" {
		j.ID = uuid.NewString()
	}
	if j.Format == "" {
		j.Format = DocumentEditFormatDOCX
	}
	if j.Mode == "" {
		j.Mode = DocumentEditModeHybrid
	}
	if j.Status == "" {
		j.Status = DocumentEditStatusQueued
	}
	if len(j.Plan) == 0 {
		j.Plan = JSON(`{}`)
	}
	if len(j.Capabilities) == 0 {
		j.Capabilities = JSON(`{}`)
	}
	return nil
}

type DocumentEditArtifact struct {
	ID         string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	JobID      string    `json:"job_id" gorm:"type:varchar(36);not null;index"`
	TenantID   uint64    `json:"tenant_id" gorm:"not null;index"`
	Kind       string    `json:"kind" gorm:"type:varchar(32);not null"`
	FileName   string    `json:"file_name" gorm:"type:varchar(1024);not null"`
	MimeType   string    `json:"mime_type" gorm:"type:varchar(255);not null"`
	StorageRef string    `json:"-" gorm:"type:text;not null"`
	SHA256     string    `json:"sha256" gorm:"type:char(64);not null"`
	Size       int64     `json:"size" gorm:"not null;default:0"`
	Metadata   JSON      `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt  time.Time `json:"created_at"`
}

// DocumentEditOperation is an immutable description of one planned edit plus
// its terminal execution state. The quote itself is not repeated here; the
// SHA256 lets auditors correlate it with the versioned plan without exposing
// more document text in the list API.
type DocumentEditOperation struct {
	ID              string                      `json:"id" gorm:"type:varchar(36);primaryKey"`
	JobID           string                      `json:"job_id" gorm:"type:varchar(36);not null;index"`
	TenantID        uint64                      `json:"tenant_id" gorm:"not null;index"`
	OperationID     string                      `json:"operation_id" gorm:"type:varchar(128);not null"`
	Kind            string                      `json:"kind" gorm:"type:varchar(32);not null"`
	Part            string                      `json:"part" gorm:"type:varchar(32);not null;default:''"`
	AnchorSHA256    string                      `json:"anchor_sha256" gorm:"type:char(64);not null"`
	ExpectedMatches int                         `json:"expected_matches" gorm:"not null;default:1"`
	ActualMatches   *int                        `json:"actual_matches,omitempty"`
	EngineName      string                      `json:"engine_name,omitempty" gorm:"type:varchar(64);not null;default:''"`
	EngineMessage   string                      `json:"engine_message,omitempty" gorm:"type:text;not null;default:''"`
	Status          DocumentEditOperationStatus `json:"status" gorm:"type:varchar(16);not null"`
	ErrorMessage    string                      `json:"error_message,omitempty" gorm:"type:text;not null;default:''"`
	CreatedAt       time.Time                   `json:"created_at"`
	AppliedAt       *time.Time                  `json:"applied_at,omitempty"`
}

func (o *DocumentEditOperation) BeforeCreate(_ *gorm.DB) error {
	if o.ID == "" {
		o.ID = uuid.NewString()
	}
	if o.Status == "" {
		o.Status = DocumentEditOperationPlanned
	}
	return nil
}

func (a *DocumentEditArtifact) BeforeCreate(_ *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if len(a.Metadata) == 0 {
		a.Metadata = JSON(`{}`)
	}
	return nil
}

type DocumentEditTaskPayload struct {
	TenantID uint64 `json:"tenant_id"`
	UserID   string `json:"user_id"`
	JobID    string `json:"job_id"`
}

type DocumentEditCreateRequest struct {
	FileName    string
	MimeType    string
	Mode        DocumentEditMode
	Instruction string
	ModelID     string
	PlanJSON    string
}

type DocumentEditStageRun struct {
	ID              string                  `json:"id" gorm:"type:varchar(36);primaryKey"`
	JobID           string                  `json:"job_id" gorm:"type:varchar(36);not null;index"`
	TenantID        uint64                  `json:"tenant_id" gorm:"not null;index"`
	Stage           string                  `json:"stage" gorm:"type:varchar(32);not null"`
	Attempt         int                     `json:"attempt" gorm:"not null;default:1"`
	EngineName      string                  `json:"engine_name,omitempty" gorm:"type:varchar(64);not null;default:''"`
	EngineVersion   string                  `json:"engine_version,omitempty" gorm:"type:varchar(64);not null;default:''"`
	ProtocolVersion string                  `json:"protocol_version,omitempty" gorm:"type:varchar(32);not null;default:''"`
	Status          DocumentEditStageStatus `json:"status" gorm:"type:varchar(16);not null"`
	StartedAt       time.Time               `json:"started_at"`
	CompletedAt     *time.Time              `json:"completed_at,omitempty"`
	DurationMS      int64                   `json:"duration_ms" gorm:"not null;default:0"`
	InputSummary    JSON                    `json:"input_summary,omitempty" gorm:"type:jsonb"`
	OutputSummary   JSON                    `json:"output_summary,omitempty" gorm:"type:jsonb"`
	ErrorCode       string                  `json:"error_code,omitempty" gorm:"type:varchar(64);not null;default:''"`
	ErrorMessage    string                  `json:"error_message,omitempty" gorm:"type:text;not null;default:''"`
	Metadata        JSON                    `json:"metadata,omitempty" gorm:"type:jsonb"`
	CreatedAt       time.Time               `json:"created_at"`
	UpdatedAt       time.Time               `json:"updated_at"`
}

func (s *DocumentEditStageRun) BeforeCreate(_ *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.Status == "" {
		s.Status = DocumentEditStageRunning
	}
	if s.Attempt <= 0 {
		s.Attempt = 1
	}
	if s.StartedAt.IsZero() {
		s.StartedAt = time.Now().UTC()
	}
	if len(s.InputSummary) == 0 {
		s.InputSummary = JSON(`{}`)
	}
	if len(s.OutputSummary) == 0 {
		s.OutputSummary = JSON(`{}`)
	}
	if len(s.Metadata) == 0 {
		s.Metadata = JSON(`{}`)
	}
	return nil
}

type DocumentEditDebugBlob struct {
	ID          string    `json:"id" gorm:"type:varchar(36);primaryKey"`
	JobID       string    `json:"job_id" gorm:"type:varchar(36);not null;index"`
	TenantID    uint64    `json:"tenant_id" gorm:"not null;index"`
	StageRunID  string    `json:"stage_run_id" gorm:"type:varchar(36);not null;index"`
	Kind        string    `json:"kind" gorm:"type:varchar(64);not null"`
	ContentType string    `json:"content_type" gorm:"type:varchar(255);not null"`
	StorageRef  string    `json:"-" gorm:"type:text;not null"`
	SHA256      string    `json:"sha256" gorm:"type:char(64);not null"`
	Size        int64     `json:"size" gorm:"not null;default:0"`
	CreatedAt   time.Time `json:"created_at"`
}

func (b *DocumentEditDebugBlob) BeforeCreate(_ *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	return nil
}

type DocumentEditDebug struct {
	Job           *DocumentEditJob         `json:"job"`
	Stages        []*DocumentEditStageRun  `json:"stages"`
	Blobs         []*DocumentEditDebugBlob `json:"blobs"`
	Model         map[string]any           `json:"model,omitempty"`
	TraceRecorded bool                     `json:"trace_recorded"`
}

type DocumentEditComparisonRequest struct {
	Modes    []DocumentEditMode             `json:"modes"`
	Strategy DocumentEditComparisonStrategy `json:"strategy"`
}

type DocumentEditComparison struct {
	GroupID string             `json:"group_id"`
	Jobs    []*DocumentEditJob `json:"jobs"`
}
