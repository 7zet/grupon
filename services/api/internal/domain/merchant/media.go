package merchant

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // registers the JPEG decoder
	_ "image/png"  // registers the PNG decoder
	"io"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "golang.org/x/image/webp" // registers the WebP decoder
)

// MaxMediaBytes is the largest accepted image, matching the API contract.
const MaxMediaBytes = 10 << 20

// ErrMediaNotFound reports a missing or foreign asset. ErrMediaInUse reports an asset still used by a deal.
var (
	ErrMediaNotFound = errors.New("merchant: media not found")
	ErrMediaInUse    = errors.New("merchant: media in use")
	ErrMediaNotReady = errors.New("merchant: media upload not stored yet")
	ErrMediaFormat   = errors.New("merchant: media content does not match its declared type")
)

var allowedMediaTypes = map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp"}

// ObjectStore is the storage port used for media bytes.
type ObjectStore interface {
	Put(ctx context.Context, key string, body io.Reader) error
	Delete(ctx context.Context, key string) error
	PublicURL(key string) string
}

// Asset is one media library entry.
type Asset struct {
	ID          uuid.UUID
	MerchantID  uuid.UUID
	Filename    string
	ContentType string
	Size        int64
	StorageKey  string
	Width       int
	Height      int
	Ready       bool
	CreatedAt   time.Time
}

// MediaRepository persists media library rows scoped to one merchant.
type MediaRepository interface {
	CreateAsset(ctx context.Context, asset Asset) (Asset, error)
	GetAsset(ctx context.Context, merchantID, assetID uuid.UUID) (Asset, error)
	MarkReady(ctx context.Context, merchantID, assetID uuid.UUID, width, height int) error
	ListAssets(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]Asset, error)
	SoftDeleteUnused(ctx context.Context, merchantID, assetID uuid.UUID) (bool, error)
}

// MediaService implements the media library: presigned-style uploads and image checks.
type MediaService struct {
	repo    MediaRepository
	objects ObjectStore
	now     func() time.Time
}

// NewMediaService binds the media rules to storage and persistence.
func NewMediaService(repo MediaRepository, objects ObjectStore, now func() time.Time) *MediaService {
	if now == nil {
		now = time.Now
	}
	return &MediaService{repo: repo, objects: objects, now: now}
}

// Upload is the reservation returned to the browser before the bytes are sent.
type Upload struct {
	Asset     Asset
	UploadURL string
}

// CreateUpload reserves an asset row and a storage key. The bytes are sent to UploadURL next.
func (s *MediaService) CreateUpload(ctx context.Context, merchantID uuid.UUID, filename, contentType string, size int64) (Upload, error) {
	ext, ok := allowedMediaTypes[contentType]
	if !ok {
		return Upload{}, &InvalidInputError{Field: "content_type", Code: "unsupported_type"}
	}
	if size < 1 || size > MaxMediaBytes {
		return Upload{}, &InvalidInputError{Field: "size", Code: "out_of_range"}
	}
	name := strings.TrimSpace(path.Base(filename))
	if name == "" || len(name) > 255 {
		return Upload{}, &InvalidInputError{Field: "filename", Code: "invalid"}
	}
	id := uuid.New()
	key := fmt.Sprintf("%s/%s.%s", id.String()[:2], id.String(), ext)
	asset, err := s.repo.CreateAsset(ctx, Asset{
		ID: id, MerchantID: merchantID, Filename: name, ContentType: contentType, Size: size, StorageKey: key,
	})
	if err != nil {
		return Upload{}, err
	}
	return Upload{Asset: asset, UploadURL: "/api/v1/merchant/media/" + asset.ID.String() + "/content"}, nil
}

// Store receives the bytes, checks that they really are the declared image type, and marks the asset ready.
func (s *MediaService) Store(ctx context.Context, merchantID, assetID uuid.UUID, body io.Reader) (Asset, error) {
	asset, err := s.repo.GetAsset(ctx, merchantID, assetID)
	if err != nil {
		return Asset{}, err
	}
	if asset.Ready {
		return Asset{}, ErrMediaNotReady
	}
	data, err := io.ReadAll(io.LimitReader(body, MaxMediaBytes+1))
	if err != nil {
		return Asset{}, fmt.Errorf("read media: %w", err)
	}
	if int64(len(data)) > MaxMediaBytes || int64(len(data)) != asset.Size {
		return Asset{}, &InvalidInputError{Field: "body", Code: "size_mismatch"}
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return Asset{}, ErrMediaFormat
	}
	if allowedMediaTypes["image/"+format] == "" || "image/"+format != asset.ContentType {
		return Asset{}, ErrMediaFormat
	}
	if err := s.objects.Put(ctx, asset.StorageKey, bytes.NewReader(data)); err != nil {
		return Asset{}, fmt.Errorf("store media: %w", err)
	}
	if err := s.repo.MarkReady(ctx, merchantID, assetID, config.Width, config.Height); err != nil {
		return Asset{}, err
	}
	asset.Width, asset.Height, asset.Ready = config.Width, config.Height, true
	return asset, nil
}

// List returns ready assets, newest first.
func (s *MediaService) List(ctx context.Context, merchantID uuid.UUID, limit, offset int) ([]Asset, error) {
	return s.repo.ListAssets(ctx, merchantID, limit, offset)
}

// Delete removes an asset the library no longer needs. Assets used by a deal are kept.
func (s *MediaService) Delete(ctx context.Context, merchantID, assetID uuid.UUID) error {
	removed, err := s.repo.SoftDeleteUnused(ctx, merchantID, assetID)
	if err != nil {
		return err
	}
	if !removed {
		return ErrMediaInUse
	}
	return nil
}

// PublicURL is the browser address of a stored asset.
func (s *MediaService) PublicURL(asset Asset) string {
	return s.objects.PublicURL(asset.StorageKey)
}
