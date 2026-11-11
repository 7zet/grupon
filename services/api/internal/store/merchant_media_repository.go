package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/dealo/dealo/services/api/internal/domain/merchant"
	sqlc "github.com/dealo/dealo/services/api/internal/store/sqlc"
)

// MerchantMediaRepository persists media library rows. Every query is scoped to one merchant.
type MerchantMediaRepository struct {
	queries *sqlc.Queries
}

// NewMerchantMediaRepository binds the media queries to a pool.
func NewMerchantMediaRepository(pool *pgxpool.Pool) *MerchantMediaRepository {
	return &MerchantMediaRepository{queries: sqlc.New(pool)}
}

var _ merchant.MediaRepository = (*MerchantMediaRepository)(nil)

func (r *MerchantMediaRepository) CreateAsset(ctx context.Context, asset merchant.Asset) (merchant.Asset, error) {
	row, err := r.queries.CreateMediaAsset(ctx, sqlc.CreateMediaAssetParams{
		MerchantID:  toPgUUID(asset.MerchantID),
		Filename:    asset.Filename,
		ContentType: asset.ContentType,
		Size:        asset.Size,
		StorageKey:  asset.StorageKey,
	})
	if err != nil {
		return merchant.Asset{}, fmt.Errorf("create media asset: %w", err)
	}
	asset.ID = toUUID(row.ID)
	asset.CreatedAt = row.CreatedAt.Time
	asset.Ready = row.Status == "ready"
	return asset, nil
}

func (r *MerchantMediaRepository) GetAsset(ctx context.Context, merchantID, assetID uuid.UUID) (merchant.Asset, error) {
	row, err := r.queries.GetMediaAssetForMerchant(ctx, sqlc.GetMediaAssetForMerchantParams{
		ID: toPgUUID(assetID), MerchantID: toPgUUID(merchantID),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return merchant.Asset{}, merchant.ErrMediaNotFound
	}
	if err != nil {
		return merchant.Asset{}, fmt.Errorf("get media asset: %w", err)
	}
	return assetFromRow(row.ID, row.MerchantID, row.Filename, row.ContentType, row.Size, row.StorageKey,
		row.Width, row.Height, row.Status, row.CreatedAt), nil
}

func (r *MerchantMediaRepository) MarkReady(ctx context.Context, merchantID, assetID uuid.UUID, width, height int) error {
	rows, err := r.queries.MarkMediaAssetReady(ctx, sqlc.MarkMediaAssetReadyParams{
		ID: toPgUUID(assetID), MerchantID: toPgUUID(merchantID), Width: pgInt4(width), Height: pgInt4(height),
	})
	if err != nil {
		return fmt.Errorf("mark media ready: %w", err)
	}
	if rows == 0 {
		return merchant.ErrMediaNotFound
	}
	return nil
}

func (r *MerchantMediaRepository) ListAssets(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]merchant.Asset, error) {
	rows, err := r.queries.ListMediaAssets(ctx, sqlc.ListMediaAssetsParams{
		MerchantID: toPgUUID(merchantID), Limit: int32(limit), Offset: int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list media assets: %w", err)
	}
	assets := make([]merchant.Asset, 0, len(rows))
	for _, row := range rows {
		assets = append(assets, assetFromRow(row.ID, row.MerchantID, row.Filename, row.ContentType, row.Size, row.StorageKey,
			row.Width, row.Height, row.Status, row.CreatedAt))
	}
	return assets, nil
}

func (r *MerchantMediaRepository) SoftDeleteUnused(ctx context.Context, merchantID, assetID uuid.UUID) (bool, error) {
	rows, err := r.queries.SoftDeleteUnusedMediaAsset(ctx, sqlc.SoftDeleteUnusedMediaAssetParams{
		ID: toPgUUID(assetID), MerchantID: toPgUUID(merchantID),
	})
	if err != nil {
		return false, fmt.Errorf("delete media asset: %w", err)
	}
	return rows > 0, nil
}

func assetFromRow(id, merchantID pgtype.UUID, filename, contentType string, size int64, key string,
	width, height pgtype.Int4, status string, createdAt pgtype.Timestamptz) merchant.Asset {
	return merchant.Asset{
		ID:          toUUID(id),
		MerchantID:  toUUID(merchantID),
		Filename:    filename,
		ContentType: contentType,
		Size:        size,
		StorageKey:  key,
		Width:       int(width.Int32),
		Height:      int(height.Int32),
		Ready:       status == "ready",
		CreatedAt:   createdAt.Time.UTC().Truncate(time.Microsecond),
	}
}

func pgInt4(value int) pgtype.Int4 {
	return pgtype.Int4{Int32: int32(value), Valid: true}
}
