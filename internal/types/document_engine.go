package types

import "context"

type DocumentEngineRequest struct {
	RequestID string
	JobID     string
	Format    DocumentEditFormat
	SHA256    string
	Document  []byte
}

type DocumentEngineArtifact struct {
	Kind     string `json:"kind"`
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`
	Content  []byte `json:"-"`
	SHA256   string `json:"sha256"`
}

type DocumentEngineResult struct {
	EngineName       string
	EngineVersion    string
	Status           string
	Text             string
	OutlineJSON      string
	ValidationJSON   string
	Artifacts        []DocumentEngineArtifact
	Warnings         []string
	OperationResults []DocumentEngineOperationResult
}

type DocumentEngineOperationResult struct {
	OperationID   string `json:"operation_id"`
	Kind          string `json:"kind"`
	Status        string `json:"status"`
	ActualMatches int    `json:"actual_matches"`
	EngineName    string `json:"engine_name"`
	Message       string `json:"message,omitempty"`
}

type DocumentEngineSearchMatch struct {
	Part    string
	Start   int
	End     int
	Quote   string
	Context string
}

type DocumentEngineCapabilities struct {
	EngineName      string             `json:"engine_name"`
	EngineVersion   string             `json:"engine_version"`
	ProtocolVersion string             `json:"protocol_version"`
	Format          DocumentEditFormat `json:"format"`
	Operations      []string           `json:"operations"`
	TrackedChanges  bool               `json:"tracked_changes"`
	Comments        bool               `json:"comments"`
	Rendering       bool               `json:"rendering"`
	Validation      bool               `json:"validation"`
}

type DocumentEngineHealth struct {
	EngineName      string `json:"engine_name"`
	EngineVersion   string `json:"engine_version"`
	ProtocolVersion string `json:"protocol_version"`
	Status          string `json:"status"`
	Message         string `json:"message"`
}

// DocumentEngine is the business-facing contract. Implementations may be a
// direct gRPC client or an orchestrator; application services never depend on
// Adeu, OfficeCLI, OOXML, or a worker filesystem.
type DocumentEngine interface {
	Capabilities(context.Context) (DocumentEngineCapabilities, error)
	Health(context.Context) (DocumentEngineHealth, error)
	Inspect(context.Context, *DocumentEngineRequest) (*DocumentEngineResult, error)
	Outline(context.Context, *DocumentEngineRequest) (*DocumentEngineResult, error)
	Search(context.Context, *DocumentEngineRequest, string, bool, bool, int) ([]DocumentEngineSearchMatch, error)
	Apply(context.Context, *DocumentEngineRequest, *EditPlan, string) (*DocumentEngineResult, error)
	Validate(context.Context, *DocumentEngineRequest) (*DocumentEngineResult, error)
	Render(context.Context, *DocumentEngineRequest, string) (*DocumentEngineResult, error)
}
