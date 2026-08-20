package interfaces

import (
	"context"
	"io"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
)

type DocumentEditRepository interface {
	Create(context.Context, *types.DocumentEditJob) error
	List(context.Context, uint64, string) ([]*types.DocumentEditJob, error)
	Get(context.Context, uint64, string, string) (*types.DocumentEditJob, error)
	Cancel(context.Context, *types.DocumentEditJob) error
	Start(context.Context, *types.DocumentEditJob) (bool, error)
	Fail(context.Context, *types.DocumentEditJob, string, string) (bool, error)
	Update(context.Context, *types.DocumentEditJob) error
	SavePlanning(context.Context, *types.DocumentEditJob) error
	RecordOperations(context.Context, *types.DocumentEditJob, []*types.DocumentEditOperation) error
	UpdateOperationStatus(context.Context, *types.DocumentEditJob, types.DocumentEditOperationStatus, string) error
	UpdateOperationResults(context.Context, *types.DocumentEditJob, []types.DocumentEngineOperationResult) error
	Complete(context.Context, *types.DocumentEditJob, []*types.DocumentEditArtifact, []*types.DocumentEditOperation) error
	CreateStage(context.Context, *types.DocumentEditStageRun) error
	UpdateStage(context.Context, *types.DocumentEditStageRun) error
	CreateDebugBlob(context.Context, *types.DocumentEditDebugBlob) error
	ListStages(context.Context, uint64, string, string) ([]*types.DocumentEditStageRun, error)
	ListDebugBlobs(context.Context, uint64, string, string) ([]*types.DocumentEditDebugBlob, error)
	GetDebugBlob(context.Context, uint64, string, string, string, string) (*types.DocumentEditDebugBlob, error)
	CreateComparisonJobs(context.Context, []*types.DocumentEditJob) error
	ListComparison(context.Context, uint64, string, string) ([]*types.DocumentEditJob, error)
}

type DocumentEngineSet struct {
	Adeu      types.DocumentEngine
	OfficeCLI types.DocumentEngine
}

type DocumentEditService interface {
	Capabilities(context.Context) (map[string]types.DocumentEngineCapabilities, error)
	Health(context.Context) (map[string]types.DocumentEngineHealth, error)
	Create(context.Context, uint64, string, types.DocumentEditCreateRequest, io.Reader) (*types.DocumentEditJob, error)
	List(context.Context, uint64, string) ([]*types.DocumentEditJob, error)
	Get(context.Context, uint64, string, string) (*types.DocumentEditJob, error)
	Cancel(context.Context, uint64, string, string) error
	OpenArtifact(context.Context, uint64, string, string, string) (*types.DocumentEditArtifact, io.ReadCloser, error)
	Debug(context.Context, uint64, string, string) (*types.DocumentEditDebug, error)
	OpenDebugBlob(context.Context, uint64, string, string, string, string) (*types.DocumentEditDebugBlob, io.ReadCloser, error)
	Compare(context.Context, uint64, string, string, types.DocumentEditComparisonRequest) (*types.DocumentEditComparison, error)
	Comparison(context.Context, uint64, string, string) (*types.DocumentEditComparison, error)
	Events(context.Context, uint64, string, string, func(*types.DocumentEditJob) error) error
	Process(context.Context, *asynq.Task) error
}
