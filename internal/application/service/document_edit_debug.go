package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

func jsonValue(value any) types.JSON {
	data, err := json.Marshal(value)
	if err != nil {
		return types.JSON(`{}`)
	}
	return types.JSON(data)
}

func documentEditAttempt(ctx context.Context) int {
	retries, ok := asynq.GetRetryCount(ctx)
	if !ok {
		return 1
	}
	return retries + 1
}

func (s *documentEditService) startStage(ctx context.Context, job *types.DocumentEditJob, stage, engine string, input any) (*types.DocumentEditStageRun, error) {
	run := &types.DocumentEditStageRun{
		JobID: job.ID, TenantID: job.TenantID, Stage: stage, Attempt: documentEditAttempt(ctx),
		EngineName: engine, Status: types.DocumentEditStageRunning, StartedAt: time.Now(), InputSummary: jsonValue(input),
	}
	if err := s.repo.CreateStage(context.WithoutCancel(ctx), run); err != nil {
		return nil, err
	}
	return run, nil
}

func enrichDocumentEditStage(ctx context.Context, run *types.DocumentEditStageRun, engine types.DocumentEngine) {
	if run == nil || engine == nil {
		return
	}
	capability, err := engine.Capabilities(ctx)
	if err != nil {
		return
	}
	run.EngineName = capability.EngineName
	run.EngineVersion = capability.EngineVersion
	run.ProtocolVersion = capability.ProtocolVersion
}

func (s *documentEditService) finishStage(ctx context.Context, run *types.DocumentEditStageRun, status types.DocumentEditStageStatus, output any, code string, err error) error {
	if run == nil {
		return nil
	}
	now := time.Now()
	run.Status = status
	run.CompletedAt = &now
	run.DurationMS = now.Sub(run.StartedAt).Milliseconds()
	run.OutputSummary = jsonValue(output)
	run.ErrorCode = code
	if err != nil {
		run.ErrorMessage = err.Error()
	}
	return s.repo.UpdateStage(context.WithoutCancel(ctx), run)
}

func (s *documentEditService) saveDebugBlob(ctx context.Context, job *types.DocumentEditJob, stage *types.DocumentEditStageRun, kind, contentType string, data []byte) error {
	if stage == nil || len(data) == 0 {
		return nil
	}
	ref, err := s.files.SaveBytes(context.WithoutCancel(ctx), data, job.TenantID, fmt.Sprintf("office-edits/%s/debug/%s-%s", job.ID, stage.ID, kind), false)
	if err != nil {
		return err
	}
	blob := &types.DocumentEditDebugBlob{
		JobID: job.ID, TenantID: job.TenantID, StageRunID: stage.ID, Kind: kind,
		ContentType: contentType, StorageRef: ref, SHA256: digest(data), Size: int64(len(data)),
	}
	if err := s.repo.CreateDebugBlob(context.WithoutCancel(ctx), blob); err != nil {
		_ = s.files.DeleteFile(context.WithoutCancel(ctx), ref)
		return err
	}
	return nil
}

func (s *documentEditService) Debug(ctx context.Context, tenantID uint64, userID, id string) (*types.DocumentEditDebug, error) {
	job, err := s.Get(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	stages, err := s.repo.ListStages(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	blobs, err := s.repo.ListDebugBlobs(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	result := &types.DocumentEditDebug{Job: job, Stages: stages, Blobs: blobs, TraceRecorded: len(stages) > 0}
	if job.ModelID != "" {
		if model, modelErr := s.models.GetModelByID(ctx, job.ModelID); modelErr == nil && model != nil {
			result.Model = map[string]any{
				"id": model.ID, "name": model.Name, "display_name": model.DisplayName,
				"source": model.Source, "type": model.Type,
			}
		}
	}
	return result, nil
}

func (s *documentEditService) OpenDebugBlob(ctx context.Context, tenantID uint64, userID, id, stageID, kind string) (*types.DocumentEditDebugBlob, io.ReadCloser, error) {
	if _, err := s.Get(ctx, tenantID, userID, id); err != nil {
		return nil, nil, err
	}
	blob, err := s.repo.GetDebugBlob(ctx, tenantID, userID, id, stageID, kind)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, ErrDocumentEditNotFound
	}
	if err != nil {
		return nil, nil, err
	}
	file, err := s.files.GetFile(ctx, blob.StorageRef)
	return blob, file, err
}

func terminalDocumentEditStatus(status types.DocumentEditStatus) bool {
	return status == types.DocumentEditStatusCompleted || status == types.DocumentEditStatusFailed || status == types.DocumentEditStatusCancelled
}

func (s *documentEditService) Compare(ctx context.Context, tenantID uint64, userID, id string, request types.DocumentEditComparisonRequest) (*types.DocumentEditComparison, error) {
	source, err := s.Get(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	if !terminalDocumentEditStatus(source.Status) {
		return nil, fmt.Errorf("%w: comparisons can only start from a terminal task", ErrDocumentEditInvalid)
	}
	if request.Strategy == "" {
		request.Strategy = types.DocumentEditComparisonReplan
	}
	if request.Strategy != types.DocumentEditComparisonReplan && request.Strategy != types.DocumentEditComparisonLockedPlan {
		return nil, fmt.Errorf("%w: unsupported comparison strategy %q", ErrDocumentEditInvalid, request.Strategy)
	}
	seen := make(map[types.DocumentEditMode]struct{})
	modes := make([]types.DocumentEditMode, 0, len(request.Modes))
	for _, rawMode := range request.Modes {
		mode, normalizeErr := s.normalizeMode(rawMode)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		if _, exists := seen[mode]; exists {
			continue
		}
		if err := s.ensureModeConfigured(mode); err != nil {
			return nil, err
		}
		seen[mode] = struct{}{}
		modes = append(modes, mode)
	}
	if len(modes) == 0 {
		return nil, fmt.Errorf("%w: at least one comparison mode is required", ErrDocumentEditInvalid)
	}
	if request.Strategy == types.DocumentEditComparisonReplan && strings.TrimSpace(source.ModelID) == "" {
		return nil, fmt.Errorf("%w: the source task has no fixed planning model", ErrDocumentEditInvalid)
	}

	var lockedPlan *types.EditPlan
	if request.Strategy == types.DocumentEditComparisonLockedPlan {
		if len(source.Plan) == 0 || string(source.Plan) == "{}" {
			return nil, fmt.Errorf("%w: the source task has no plan to lock", ErrDocumentEditInvalid)
		}
		lockedPlan = &types.EditPlan{}
		if err := json.Unmarshal(source.Plan, lockedPlan); err != nil {
			return nil, fmt.Errorf("%w: source plan is invalid: %v", ErrDocumentEditInvalid, err)
		}
		// Preflight every selected mode before creating any task.
		for _, mode := range modes {
			if err := validateEditPlan(lockedPlan, source.SourceSHA256, mode); err != nil {
				return nil, err
			}
		}
	}

	data, err := s.readSource(ctx, source)
	if err != nil {
		return nil, err
	}
	groupID := strings.TrimSpace(source.ComparisonGroupID)
	if groupID == "" {
		groupID = uuid.NewString()
		source.ComparisonGroupID = groupID
		if err := s.repo.Update(ctx, source); err != nil {
			return nil, err
		}
	}

	jobs := make([]*types.DocumentEditJob, 0, len(modes))
	savedRefs := make([]string, 0, len(modes))
	for _, mode := range modes {
		job := &types.DocumentEditJob{
			ID: uuid.NewString(), TenantID: tenantID, UserID: userID, Format: source.Format, Mode: mode,
			Status: types.DocumentEditStatusQueued, FileName: source.FileName, MimeType: source.MimeType,
			FileSize: source.FileSize, SourceSHA256: source.SourceSHA256, Instruction: source.Instruction,
			ModelID: source.ModelID, Plan: types.JSON(`{}`), Capabilities: types.JSON(`{}`),
			ComparisonGroupID: groupID, ComparisonParentID: source.ID, ComparisonStrategy: request.Strategy,
		}
		if lockedPlan != nil {
			job.Plan = append(types.JSON(nil), source.Plan...)
		}
		job.SourceRef, err = s.files.SaveBytes(ctx, data, tenantID, fmt.Sprintf("office-edits/%s/source.docx", job.ID), false)
		if err != nil {
			for _, ref := range savedRefs {
				_ = s.files.DeleteFile(ctx, ref)
			}
			return nil, err
		}
		savedRefs = append(savedRefs, job.SourceRef)
		jobs = append(jobs, job)
	}
	if err := s.repo.CreateComparisonJobs(ctx, jobs); err != nil {
		for _, ref := range savedRefs {
			_ = s.files.DeleteFile(ctx, ref)
		}
		return nil, err
	}
	for _, job := range jobs {
		if lockedPlan != nil {
			if err := s.repo.RecordOperations(ctx, job, operationLedger(job, lockedPlan, types.DocumentEditOperationPlanned)); err != nil {
				job.Status, job.ErrorCode, job.ErrorMessage = types.DocumentEditStatusFailed, "operation_ledger_failed", err.Error()
				_ = s.repo.Update(ctx, job)
				continue
			}
		}
		payload, _ := json.Marshal(types.DocumentEditTaskPayload{TenantID: tenantID, UserID: userID, JobID: job.ID})
		queue, _ := types.QueueForTaskType(types.TypeDocumentEditProcess)
		if _, enqueueErr := s.tasks.Enqueue(asynq.NewTask(types.TypeDocumentEditProcess, payload), asynq.Queue(queue), asynq.MaxRetry(2), asynq.Timeout(30*time.Minute)); enqueueErr != nil {
			job.Status, job.ErrorCode, job.ErrorMessage = types.DocumentEditStatusFailed, "enqueue_failed", enqueueErr.Error()
			_ = s.repo.Update(ctx, job)
		}
	}
	return s.Comparison(ctx, tenantID, userID, source.ID)
}

func (s *documentEditService) Comparison(ctx context.Context, tenantID uint64, userID, id string) (*types.DocumentEditComparison, error) {
	job, err := s.Get(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	groupID := job.ComparisonGroupID
	if groupID == "" {
		return &types.DocumentEditComparison{Jobs: []*types.DocumentEditJob{job}}, nil
	}
	jobs, err := s.repo.ListComparison(ctx, tenantID, userID, groupID)
	if err != nil {
		return nil, err
	}
	return &types.DocumentEditComparison{GroupID: groupID, Jobs: jobs}, nil
}
