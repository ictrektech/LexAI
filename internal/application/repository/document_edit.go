package repository

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

type documentEditRepository struct {
	db *gorm.DB
}

func NewDocumentEditRepository(db *gorm.DB) interfaces.DocumentEditRepository {
	return &documentEditRepository{db: db}
}

func (r *documentEditRepository) Create(ctx context.Context, job *types.DocumentEditJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *documentEditRepository) List(ctx context.Context, tenantID uint64, userID string) ([]*types.DocumentEditJob, error) {
	var rows []*types.DocumentEditJob
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Preload("Artifacts", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Operations", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Order("updated_at DESC").Find(&rows).Error
	return rows, err
}

func (r *documentEditRepository) Get(ctx context.Context, tenantID uint64, userID, id string) (*types.DocumentEditJob, error) {
	var row types.DocumentEditJob
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, userID, id).
		Preload("Artifacts", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Operations", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		First(&row).Error
	return &row, err
}

func (r *documentEditRepository) Update(ctx context.Context, job *types.DocumentEditJob) error {
	result := r.db.WithContext(ctx).Model(&types.DocumentEditJob{}).
		Where("tenant_id = ? AND user_id = ? AND id = ?", job.TenantID, job.UserID, job.ID).
		Select("format", "mode", "status", "file_name", "mime_type", "file_size", "source_sha256", "source_ref", "instruction", "model_id", "plan", "capabilities", "comparison_group_id", "comparison_parent_id", "comparison_strategy", "error_code", "error_message", "started_at", "completed_at", "updated_at").
		Updates(job)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *documentEditRepository) SavePlanning(ctx context.Context, job *types.DocumentEditJob) error {
	result := r.db.WithContext(ctx).Model(&types.DocumentEditJob{}).
		Where("tenant_id = ? AND user_id = ? AND id = ? AND status = ?", job.TenantID, job.UserID, job.ID, types.DocumentEditStatusRunning).
		Updates(map[string]any{
			"model_id": job.ModelID, "plan": job.Plan, "capabilities": job.Capabilities, "updated_at": time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *documentEditRepository) Cancel(ctx context.Context, job *types.DocumentEditJob) error {
	now := time.Now()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&types.DocumentEditJob{}).
			Where("tenant_id = ? AND user_id = ? AND id = ? AND status IN ?", job.TenantID, job.UserID, job.ID, []types.DocumentEditStatus{types.DocumentEditStatusQueued, types.DocumentEditStatusRunning}).
			Updates(map[string]any{
				"status":        types.DocumentEditStatusCancelled,
				"error_code":    "cancelled",
				"error_message": "document edit task cancelled",
				"completed_at":  &now,
				"updated_at":    &now,
			})
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		return tx.Model(&types.DocumentEditOperation{}).
			Where("tenant_id = ? AND job_id = ?", job.TenantID, job.ID).
			Updates(map[string]any{"status": types.DocumentEditOperationCancelled, "error_message": "document edit task cancelled"}).Error
	})
}

func (r *documentEditRepository) Start(ctx context.Context, job *types.DocumentEditJob) (bool, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&types.DocumentEditJob{}).
		Where("tenant_id = ? AND user_id = ? AND id = ? AND status IN ?", job.TenantID, job.UserID, job.ID, []types.DocumentEditStatus{types.DocumentEditStatusQueued, types.DocumentEditStatusFailed}).
		Updates(map[string]any{
			"status":        types.DocumentEditStatusRunning,
			"error_code":    "",
			"error_message": "",
			"started_at":    &now,
			"completed_at":  nil,
			"updated_at":    &now,
		})
	return result.RowsAffected > 0, result.Error
}

func (r *documentEditRepository) Fail(ctx context.Context, job *types.DocumentEditJob, code, message string) (bool, error) {
	now := time.Now()
	returnValue := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&types.DocumentEditJob{}).
			Where("tenant_id = ? AND user_id = ? AND id = ? AND status IN ?", job.TenantID, job.UserID, job.ID, []types.DocumentEditStatus{types.DocumentEditStatusQueued, types.DocumentEditStatusRunning}).
			Updates(map[string]any{
				"status":        types.DocumentEditStatusFailed,
				"error_code":    code,
				"error_message": message,
				"plan":          job.Plan,
				"capabilities":  job.Capabilities,
				"model_id":      job.ModelID,
				"started_at":    job.StartedAt,
				"completed_at":  &now,
				"updated_at":    &now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		returnValue = true
		return tx.Model(&types.DocumentEditOperation{}).
			Where("tenant_id = ? AND job_id = ? AND status = ?", job.TenantID, job.ID, types.DocumentEditOperationPlanned).
			Updates(map[string]any{"status": types.DocumentEditOperationFailed, "error_message": message}).Error
	})
	return returnValue, err
}

func (r *documentEditRepository) RecordOperations(ctx context.Context, job *types.DocumentEditJob, operations []*types.DocumentEditOperation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lock := tx.Model(&types.DocumentEditJob{}).
			Where("tenant_id = ? AND user_id = ? AND id = ? AND status IN ?", job.TenantID, job.UserID, job.ID, []types.DocumentEditStatus{types.DocumentEditStatusQueued, types.DocumentEditStatusRunning}).
			UpdateColumn("updated_at", gorm.Expr("updated_at"))
		if lock.Error != nil {
			return lock.Error
		}
		if lock.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Where("tenant_id = ? AND job_id = ?", job.TenantID, job.ID).Delete(&types.DocumentEditOperation{}).Error; err != nil {
			return err
		}
		if len(operations) == 0 {
			return nil
		}
		return tx.Create(&operations).Error
	})
}

func (r *documentEditRepository) UpdateOperationStatus(ctx context.Context, job *types.DocumentEditJob, status types.DocumentEditOperationStatus, message string) error {
	values := map[string]any{"status": status, "error_message": message}
	if status == types.DocumentEditOperationApplied {
		now := time.Now()
		values["applied_at"] = &now
	}
	return r.db.WithContext(ctx).Model(&types.DocumentEditOperation{}).
		Where("tenant_id = ? AND job_id = ?", job.TenantID, job.ID).
		Updates(values).Error
}

func (r *documentEditRepository) UpdateOperationResults(ctx context.Context, job *types.DocumentEditJob, results []types.DocumentEngineOperationResult) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, operation := range results {
			actualMatches := operation.ActualMatches
			values := map[string]any{
				"actual_matches": &actualMatches,
				"engine_name":    operation.EngineName,
				"engine_message": operation.Message,
			}
			switch operation.Status {
			case "applied":
				values["status"] = types.DocumentEditOperationApplied
				now := time.Now()
				values["applied_at"] = &now
			case "failed":
				values["status"] = types.DocumentEditOperationFailed
				values["error_message"] = operation.Message
			}
			result := tx.Model(&types.DocumentEditOperation{}).
				Where("tenant_id = ? AND job_id = ? AND operation_id = ?", job.TenantID, job.ID, operation.OperationID).
				Updates(values)
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}

func (r *documentEditRepository) Complete(ctx context.Context, job *types.DocumentEditJob, artifacts []*types.DocumentEditArtifact, operations []*types.DocumentEditOperation) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&types.DocumentEditJob{}).
			Where("tenant_id = ? AND user_id = ? AND id = ? AND status IN ?", job.TenantID, job.UserID, job.ID, []types.DocumentEditStatus{types.DocumentEditStatusQueued, types.DocumentEditStatusRunning}).
			Select("status", "plan", "capabilities", "error_code", "error_message", "started_at", "completed_at", "updated_at").
			Updates(job)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if len(artifacts) > 0 {
			if err := tx.Create(&artifacts).Error; err != nil {
				return err
			}
		}
		if len(operations) > 0 {
			now := time.Now()
			if err := tx.Model(&types.DocumentEditOperation{}).
				Where("tenant_id = ? AND job_id = ? AND status = ?", job.TenantID, job.ID, types.DocumentEditOperationPlanned).
				Updates(map[string]any{"status": types.DocumentEditOperationApplied, "error_message": "", "applied_at": &now}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *documentEditRepository) CreateStage(ctx context.Context, stage *types.DocumentEditStageRun) error {
	return r.db.WithContext(ctx).Create(stage).Error
}

func (r *documentEditRepository) UpdateStage(ctx context.Context, stage *types.DocumentEditStageRun) error {
	result := r.db.WithContext(ctx).Model(&types.DocumentEditStageRun{}).
		Where("tenant_id = ? AND job_id = ? AND id = ?", stage.TenantID, stage.JobID, stage.ID).
		Select("engine_name", "engine_version", "protocol_version", "status", "completed_at", "duration_ms", "input_summary", "output_summary", "error_code", "error_message", "metadata", "updated_at").
		Updates(stage)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *documentEditRepository) CreateDebugBlob(ctx context.Context, blob *types.DocumentEditDebugBlob) error {
	return r.db.WithContext(ctx).Create(blob).Error
}

func (r *documentEditRepository) ListStages(ctx context.Context, tenantID uint64, userID, jobID string) ([]*types.DocumentEditStageRun, error) {
	var rows []*types.DocumentEditStageRun
	err := r.db.WithContext(ctx).Table("document_edit_stage_runs AS stages").
		Joins("JOIN document_edit_jobs AS jobs ON jobs.id = stages.job_id AND jobs.deleted_at IS NULL").
		Where("jobs.tenant_id = ? AND jobs.user_id = ? AND jobs.id = ?", tenantID, userID, jobID).
		Order("stages.started_at ASC, stages.created_at ASC").Find(&rows).Error
	return rows, err
}

func (r *documentEditRepository) ListDebugBlobs(ctx context.Context, tenantID uint64, userID, jobID string) ([]*types.DocumentEditDebugBlob, error) {
	var rows []*types.DocumentEditDebugBlob
	err := r.db.WithContext(ctx).Table("document_edit_debug_blobs AS blobs").
		Joins("JOIN document_edit_jobs AS jobs ON jobs.id = blobs.job_id AND jobs.deleted_at IS NULL").
		Where("jobs.tenant_id = ? AND jobs.user_id = ? AND jobs.id = ?", tenantID, userID, jobID).
		Order("blobs.created_at ASC").Find(&rows).Error
	return rows, err
}

func (r *documentEditRepository) GetDebugBlob(ctx context.Context, tenantID uint64, userID, jobID, stageID, kind string) (*types.DocumentEditDebugBlob, error) {
	var row types.DocumentEditDebugBlob
	err := r.db.WithContext(ctx).Table("document_edit_debug_blobs AS blobs").
		Joins("JOIN document_edit_jobs AS jobs ON jobs.id = blobs.job_id AND jobs.deleted_at IS NULL").
		Where("jobs.tenant_id = ? AND jobs.user_id = ? AND jobs.id = ? AND blobs.stage_run_id = ? AND blobs.kind = ?", tenantID, userID, jobID, stageID, kind).
		First(&row).Error
	return &row, err
}

func (r *documentEditRepository) CreateComparisonJobs(ctx context.Context, jobs []*types.DocumentEditJob) error {
	if len(jobs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return tx.Create(&jobs).Error })
}

func (r *documentEditRepository) ListComparison(ctx context.Context, tenantID uint64, userID, groupID string) ([]*types.DocumentEditJob, error) {
	var rows []*types.DocumentEditJob
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND comparison_group_id = ?", tenantID, userID, groupID).
		Preload("Artifacts", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Preload("Operations", func(db *gorm.DB) *gorm.DB { return db.Order("created_at ASC") }).
		Order("created_at ASC").Find(&rows).Error
	return rows, err
}
