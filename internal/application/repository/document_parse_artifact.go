package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type documentParseArtifactRepository struct{ db *gorm.DB }

func NewDocumentParseArtifactRepository(db *gorm.DB) interfaces.DocumentParseArtifactRepository {
	return &documentParseArtifactRepository{db: db}
}

func (r *documentParseArtifactRepository) GetByID(ctx context.Context, tenantID uint64, id string) (*types.DocumentParseArtifact, error) {
	var row types.DocumentParseArtifact
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id).First(&row).Error
	return &row, err
}

func (r *documentParseArtifactRepository) GetByFingerprint(ctx context.Context, tenantID uint64, fileHash, parserVersion string) (*types.DocumentParseArtifact, error) {
	var row types.DocumentParseArtifact
	err := r.db.WithContext(ctx).Where("tenant_id = ? AND file_hash = ? AND parser_version = ?", tenantID, fileHash, parserVersion).First(&row).Error
	return &row, err
}

func (r *documentParseArtifactRepository) Upsert(ctx context.Context, row *types.DocumentParseArtifact) error {
	if row == nil {
		return errors.New("document parse artifact is nil")
	}
	if row.ParserVersion == "" {
		row.ParserVersion = types.DocumentParseArtifactVersion
	}
	if row.Result == nil {
		row.Result = types.JSON(`{}`)
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_id"}, {Name: "file_hash"}, {Name: "parser_version"}},
		DoUpdates: clause.Assignments(map[string]any{
			"source_document_id": row.SourceDocumentID,
			"file_name":          row.FileName,
			"file_type":          row.FileType,
			"markdown_content":   row.MarkdownContent,
			"result":             row.Result,
			"updated_at":         gorm.Expr("CURRENT_TIMESTAMP"),
		}),
	}).Create(row).Error; err != nil {
		return err
	}
	// OnConflict updates the existing row but does not portably return its
	// primary key on both PostgreSQL and SQLite. Reload by the unique
	// fingerprint so callers always receive the durable artifact ID rather than
	// a freshly generated UUID that was discarded by the conflict clause.
	var stored types.DocumentParseArtifact
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND file_hash = ? AND parser_version = ?", row.TenantID, row.FileHash, row.ParserVersion).First(&stored).Error; err != nil {
		return err
	}
	*row = stored
	return nil
}

func (r *documentParseArtifactRepository) DeleteBySourceDocument(ctx context.Context, tenantID uint64, sourceDocumentID string) error {
	if strings.TrimSpace(sourceDocumentID) == "" {
		return nil
	}
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND source_document_id = ?", tenantID, sourceDocumentID).
		Delete(&types.DocumentParseArtifact{}).Error
}
