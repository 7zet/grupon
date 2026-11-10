package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dealo/dealo/services/api/internal/domain/merchant"
	sqlc "github.com/dealo/dealo/services/api/internal/store/sqlc"
)

// MerchantDocumentRepository implements merchant.DocumentRepository on PostgreSQL.
type MerchantDocumentRepository struct {
	pool *pgxpool.Pool
}

// NewMerchantDocumentRepository returns the document store.
func NewMerchantDocumentRepository(pool *pgxpool.Pool) *MerchantDocumentRepository {
	return &MerchantDocumentRepository{pool: pool}
}

// OwnedBusiness returns the owner's business id and review status, or ErrNoBusiness.
func (r *MerchantDocumentRepository) OwnedBusiness(ctx context.Context, ownerID uuid.UUID) (uuid.UUID, string, error) {
	row, err := sqlc.New(r.pool).OwnedMerchantReviewStatus(ctx, toPgUUID(ownerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", merchant.ErrNoBusiness
	}
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("load owned business: %w", err)
	}
	return toUUID(row.ID), row.ReviewStatus, nil
}

// SaveDocument stores the metadata and returns the key of the file it replaced, if any.
func (r *MerchantDocumentRepository) SaveDocument(ctx context.Context, merchantID, uploaderID uuid.UUID, doc merchant.Document) (string, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin document save: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	queries := sqlc.New(tx)
	previous, err := queries.GetMerchantDocumentKey(ctx, sqlc.GetMerchantDocumentKeyParams{
		MerchantID: toPgUUID(merchantID),
		Kind:       doc.Kind,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("load previous document: %w", err)
	}
	if err := queries.UpsertMerchantDocument(ctx, sqlc.UpsertMerchantDocumentParams{
		MerchantID:  toPgUUID(merchantID),
		Kind:        doc.Kind,
		StorageKey:  doc.StorageKey,
		ContentType: doc.ContentType,
		SizeBytes:   int32(doc.SizeBytes),
		UploadedBy:  toPgUUID(uploaderID),
	}); err != nil {
		return "", fmt.Errorf("save document: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit document save: %w", err)
	}
	return previous, nil
}

// Documents lists the uploaded documents of a business.
func (r *MerchantDocumentRepository) Documents(ctx context.Context, merchantID uuid.UUID) ([]merchant.Document, error) {
	rows, err := sqlc.New(r.pool).ListMerchantDocuments(ctx, toPgUUID(merchantID))
	if err != nil {
		return nil, fmt.Errorf("list documents: %w", err)
	}
	out := make([]merchant.Document, 0, len(rows))
	for _, row := range rows {
		out = append(out, documentFromRow(row.Kind, row.ContentType, row.SizeBytes, row.UploadedAt, row.StorageKey))
	}
	return out, nil
}

// Document returns one document, or ErrDocumentNotFound.
func (r *MerchantDocumentRepository) Document(ctx context.Context, merchantID uuid.UUID, kind string) (merchant.Document, error) {
	rows, err := sqlc.New(r.pool).ListMerchantDocuments(ctx, toPgUUID(merchantID))
	if err != nil {
		return merchant.Document{}, fmt.Errorf("load document: %w", err)
	}
	for _, row := range rows {
		if row.Kind == kind {
			return documentFromRow(row.Kind, row.ContentType, row.SizeBytes, row.UploadedAt, row.StorageKey), nil
		}
	}
	return merchant.Document{}, merchant.ErrDocumentNotFound
}

func documentFromRow(kind, contentType string, size int32, uploadedAt pgtype.Timestamptz, key string) merchant.Document {
	return merchant.Document{
		Kind:        kind,
		ContentType: contentType,
		SizeBytes:   int64(size),
		UploadedAt:  uploadedAt.Time,
		StorageKey:  key,
	}
}
