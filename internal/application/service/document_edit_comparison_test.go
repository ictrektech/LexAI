package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/config"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type comparisonFileService struct {
	files map[string][]byte
	saves int
}

func (f *comparisonFileService) CheckConnectivity(context.Context) error { return nil }
func (f *comparisonFileService) SaveFile(context.Context, *multipart.FileHeader, uint64, string) (string, error) {
	return "", nil
}
func (f *comparisonFileService) SaveBytes(_ context.Context, data []byte, _ uint64, name string, _ bool) (string, error) {
	f.saves++
	ref := fmt.Sprintf("memory://%d/%s", f.saves, name)
	f.files[ref] = append([]byte(nil), data...)
	return ref, nil
}
func (f *comparisonFileService) GetFile(_ context.Context, ref string) (io.ReadCloser, error) {
	data, ok := f.files[ref]
	if !ok {
		return nil, fmt.Errorf("missing %s", ref)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}
func (f *comparisonFileService) GetFileURL(context.Context, string) (string, error) { return "", nil }
func (f *comparisonFileService) DeleteFile(_ context.Context, ref string) error {
	delete(f.files, ref)
	return nil
}
func (f *comparisonFileService) CopyFile(context.Context, string, uint64, string) (string, error) {
	return "", nil
}

type comparisonTaskEnqueuer struct{ count int }

func (q *comparisonTaskEnqueuer) Enqueue(_ *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	q.count++
	return &asynq.TaskInfo{}, nil
}

func newComparisonTestService(t *testing.T, sourcePlan types.JSON) (*documentEditService, *types.DocumentEditJob, *comparisonFileService, *comparisonTaskEnqueuer) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.DocumentEditJob{}, &types.DocumentEditArtifact{}, &types.DocumentEditOperation{},
		&types.DocumentEditStageRun{}, &types.DocumentEditDebugBlob{},
	))
	repo := repository.NewDocumentEditRepository(db)
	files := &comparisonFileService{files: make(map[string][]byte)}
	tasks := &comparisonTaskEnqueuer{}
	sourceBytes := []byte("PK\x03\x04comparison-source")
	sourceRef := "memory://source"
	files.files[sourceRef] = sourceBytes
	job := &types.DocumentEditJob{
		ID: uuid.NewString(), TenantID: 71, UserID: "creator", Format: types.DocumentEditFormatDOCX,
		Mode: types.DocumentEditModeAdeu, Status: types.DocumentEditStatusCompleted, FileName: "contract.docx",
		MimeType: "application/vnd.openxmlformats-officedocument.wordprocessingml.document", FileSize: int64(len(sourceBytes)),
		SourceSHA256: digest(sourceBytes), SourceRef: sourceRef, Instruction: "change payment term", ModelID: "fixed-model",
		Plan: sourcePlan, Capabilities: types.JSON(`{}`),
	}
	require.NoError(t, repo.Create(context.Background(), job))
	service := &documentEditService{
		repo: repo, files: files, tasks: tasks,
		config: &config.Config{OfficeEngine: &config.OfficeEngineConfig{
			Enabled: true, Mode: "hybrid", AdeuAddr: "127.0.0.1:50052", OfficeCLIAddr: "127.0.0.1:50053",
		}},
	}
	return service, job, files, tasks
}

func TestLockedPlanComparisonRejectsAllJobsBeforeCreation(t *testing.T) {
	plan := types.JSON(`{"schema_version":"1.0","format":"DOCX","base_sha256":"placeholder","apply_mode":"atomic","operations":[{"operation_id":"op-1","kind":"add_comment","target":{"quote":"payment","expected_matches":1},"payload":{"comment":"review"}}]}`)
	service, job, files, tasks := newComparisonTestService(t, plan)
	var parsed types.EditPlan
	require.NoError(t, json.Unmarshal(job.Plan, &parsed))
	parsed.BaseSHA256 = job.SourceSHA256
	job.Plan = jsonValue(parsed)
	require.NoError(t, service.repo.Update(context.Background(), job))

	_, err := service.Compare(context.Background(), job.TenantID, job.UserID, job.ID, types.DocumentEditComparisonRequest{
		Modes:    []types.DocumentEditMode{types.DocumentEditModeAdeu, types.DocumentEditModeOfficeCLI},
		Strategy: types.DocumentEditComparisonLockedPlan,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not support comments")
	require.Zero(t, files.saves)
	require.Zero(t, tasks.count)
	stored, getErr := service.repo.Get(context.Background(), job.TenantID, job.UserID, job.ID)
	require.NoError(t, getErr)
	require.Empty(t, stored.ComparisonGroupID)
}

func TestReplanComparisonClonesFixedInputAndEnqueuesEachMode(t *testing.T) {
	service, job, files, tasks := newComparisonTestService(t, types.JSON(`{}`))
	comparison, err := service.Compare(context.Background(), job.TenantID, job.UserID, job.ID, types.DocumentEditComparisonRequest{
		Modes:    []types.DocumentEditMode{types.DocumentEditModeAdeu, types.DocumentEditModeOfficeCLI},
		Strategy: types.DocumentEditComparisonReplan,
	})
	require.NoError(t, err)
	require.NotEmpty(t, comparison.GroupID)
	require.Len(t, comparison.Jobs, 3)
	require.Equal(t, 2, files.saves)
	require.Equal(t, 2, tasks.count)
	encoded, marshalErr := json.Marshal(comparison)
	require.NoError(t, marshalErr)
	require.NotContains(t, string(encoded), "memory://")
	require.NotContains(t, string(encoded), "source_ref")
	for _, compared := range comparison.Jobs {
		if compared.ID == job.ID {
			continue
		}
		require.Equal(t, job.SourceSHA256, compared.SourceSHA256)
		require.Equal(t, job.Instruction, compared.Instruction)
		require.Equal(t, job.ModelID, compared.ModelID)
		require.Equal(t, types.DocumentEditComparisonReplan, compared.ComparisonStrategy)
		require.JSONEq(t, `{}`, string(compared.Plan))
	}
}
