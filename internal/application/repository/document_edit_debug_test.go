package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newDocumentEditDebugTestRepository(t *testing.T) (*documentEditRepository, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.DocumentEditJob{}, &types.DocumentEditArtifact{}, &types.DocumentEditOperation{},
		&types.DocumentEditStageRun{}, &types.DocumentEditDebugBlob{},
	))
	return &documentEditRepository{db: db}, db
}

func TestDocumentEditDebugOwnerIsolationAndStageLifecycle(t *testing.T) {
	repo, db := newDocumentEditDebugTestRepository(t)
	ctx := context.Background()
	job := &types.DocumentEditJob{
		ID: uuid.NewString(), TenantID: 41, UserID: "creator", Format: types.DocumentEditFormatDOCX,
		Mode: types.DocumentEditModeHybrid, Status: types.DocumentEditStatusRunning, FileName: "contract.docx",
		SourceSHA256: fmt.Sprintf("%064d", 1), SourceRef: "memory://source", Instruction: "edit",
	}
	require.NoError(t, repo.Create(ctx, job))

	stage := &types.DocumentEditStageRun{JobID: job.ID, TenantID: job.TenantID, Stage: "Inspect", EngineName: "adeu"}
	require.NoError(t, repo.CreateStage(ctx, stage))
	completed := time.Now()
	stage.Status = types.DocumentEditStageCompleted
	stage.CompletedAt = &completed
	stage.DurationMS = 12
	stage.OutputSummary = types.JSON(`{"characters":42}`)
	require.NoError(t, repo.UpdateStage(ctx, stage))

	blob := &types.DocumentEditDebugBlob{
		JobID: job.ID, TenantID: job.TenantID, StageRunID: stage.ID, Kind: "inspect_text",
		ContentType: "text/plain", StorageRef: "memory://debug", SHA256: fmt.Sprintf("%064d", 2), Size: 42,
	}
	require.NoError(t, repo.CreateDebugBlob(ctx, blob))

	stages, err := repo.ListStages(ctx, job.TenantID, job.UserID, job.ID)
	require.NoError(t, err)
	require.Len(t, stages, 1)
	require.Equal(t, types.DocumentEditStageCompleted, stages[0].Status)
	require.EqualValues(t, 12, stages[0].DurationMS)

	otherStages, err := repo.ListStages(ctx, job.TenantID, "other-user", job.ID)
	require.NoError(t, err)
	require.Empty(t, otherStages)
	_, err = repo.GetDebugBlob(ctx, job.TenantID, "other-user", job.ID, stage.ID, blob.Kind)
	require.True(t, errors.Is(err, gorm.ErrRecordNotFound))

	ownedBlob, err := repo.GetDebugBlob(ctx, job.TenantID, job.UserID, job.ID, stage.ID, blob.Kind)
	require.NoError(t, err)
	require.Equal(t, "memory://debug", ownedBlob.StorageRef)

	var stored types.DocumentEditStageRun
	require.NoError(t, db.First(&stored, "id = ?", stage.ID).Error)
	require.Equal(t, types.DocumentEditStageCompleted, stored.Status)
}

func TestSavePlanningPersistsBeforeTerminalState(t *testing.T) {
	repo, _ := newDocumentEditDebugTestRepository(t)
	ctx := context.Background()
	job := &types.DocumentEditJob{
		ID: uuid.NewString(), TenantID: 42, UserID: "creator", Format: types.DocumentEditFormatDOCX,
		Mode: types.DocumentEditModeAdeu, Status: types.DocumentEditStatusRunning, FileName: "contract.docx",
		SourceSHA256: fmt.Sprintf("%064d", 3), SourceRef: "memory://source", Instruction: "edit",
	}
	require.NoError(t, repo.Create(ctx, job))
	job.ModelID = "model-fixed"
	job.Plan = types.JSON(`{"schema_version":"1.0"}`)
	job.Capabilities = types.JSON(`{"adeu":{"protocol_version":"0.1.0"}}`)
	require.NoError(t, repo.SavePlanning(ctx, job))

	stored, err := repo.Get(ctx, job.TenantID, job.UserID, job.ID)
	require.NoError(t, err)
	require.Equal(t, types.DocumentEditStatusRunning, stored.Status)
	require.Equal(t, "model-fixed", stored.ModelID)
	require.JSONEq(t, string(job.Plan), string(stored.Plan))
}

func TestStartKeepsLifecycleTimestampsOnTheInMemoryJob(t *testing.T) {
	repo, db := newDocumentEditDebugTestRepository(t)
	ctx := context.Background()
	job := &types.DocumentEditJob{
		ID: uuid.NewString(), TenantID: 43, UserID: "creator", Format: types.DocumentEditFormatDOCX,
		Mode: types.DocumentEditModeHybrid, Status: types.DocumentEditStatusQueued, FileName: "contract.docx",
		SourceSHA256: fmt.Sprintf("%064d", 4), SourceRef: "memory://source", Instruction: "edit",
	}
	require.NoError(t, repo.Create(ctx, job))

	started, err := repo.Start(ctx, job)
	require.NoError(t, err)
	require.True(t, started)
	require.Equal(t, types.DocumentEditStatusRunning, job.Status)
	require.NotNil(t, job.StartedAt)
	require.Equal(t, time.UTC, job.StartedAt.Location())

	completed := time.Now().UTC()
	job.Status = types.DocumentEditStatusCompleted
	job.CompletedAt = &completed
	require.NoError(t, repo.Complete(ctx, job, nil, nil))

	var stored types.DocumentEditJob
	require.NoError(t, db.First(&stored, "id = ?", job.ID).Error)
	require.NotNil(t, stored.StartedAt)
	require.NotNil(t, stored.CompletedAt)
	require.WithinDuration(t, *job.StartedAt, *stored.StartedAt, time.Millisecond)
}
