package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/dealo/dealo/services/api/internal/domain/merchant"
	"github.com/dealo/dealo/services/api/internal/http/gen"
)

// MerchantMediaAPI serves the media library. Owners and managers upload; cashiers cannot.
type MerchantMediaAPI struct {
	gen.Unimplemented

	media     *merchant.MediaService
	workspace *merchant.WorkspaceService
	auth      *AuthAPI
	logger    *slog.Logger
}

var _ gen.ServerInterface = (*MerchantMediaAPI)(nil)

// NewMerchantMediaAPI binds the media service to the HTTP contract.
func NewMerchantMediaAPI(media *merchant.MediaService, workspace *merchant.WorkspaceService, auth *AuthAPI, logger *slog.Logger) *MerchantMediaAPI {
	return &MerchantMediaAPI{media: media, workspace: workspace, auth: auth, logger: logger}
}

func (a *MerchantMediaAPI) scope(w http.ResponseWriter, r *http.Request, mayEdit bool) (uuid.UUID, bool) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return uuid.Nil, false
	}
	merchantID, _, err := a.workspace.Authorize(r.Context(), account.ID, mayEdit)
	if err != nil {
		a.fail(w, r, err)
		return uuid.Nil, false
	}
	return merchantID, true
}

func (a *MerchantMediaAPI) ListMerchantMedia(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := a.scope(w, r, false)
	if !ok {
		return
	}
	assets, err := a.media.List(r.Context(), merchantID, 100, 0)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.MediaAsset, 0, len(assets))
	for _, asset := range assets {
		items = append(items, a.mapAsset(asset))
	}
	writeJSON(w, http.StatusOK, gen.MediaList{Items: items})
}

func (a *MerchantMediaAPI) CreateMerchantMediaUpload(w http.ResponseWriter, r *http.Request) {
	merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	var body gen.MediaUploadRequest
	if _, ok := decodeBody(w, r, &body); !ok {
		return
	}
	upload, err := a.media.CreateUpload(r.Context(), merchantID, body.Filename, string(body.ContentType), body.Size)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, gen.MediaUpload{
		Asset:     a.mapAsset(upload.Asset),
		UploadUrl: upload.UploadURL,
		Headers:   map[string]string{},
	})
}

func (a *MerchantMediaAPI) UploadMerchantMediaContent(w http.ResponseWriter, r *http.Request, id gen.Id) {
	merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	body := http.MaxBytesReader(w, r.Body, merchant.MaxMediaBytes+1)
	defer body.Close()
	asset, err := a.media.Store(r.Context(), merchantID, uuid.UUID(id), body)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	_ = asset
	w.WriteHeader(http.StatusNoContent)
}

func (a *MerchantMediaAPI) DeleteMerchantMedia(w http.ResponseWriter, r *http.Request, id gen.Id) {
	merchantID, ok := a.scope(w, r, true)
	if !ok {
		return
	}
	if err := a.media.Delete(r.Context(), merchantID, uuid.UUID(id)); err != nil {
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *MerchantMediaAPI) mapAsset(asset merchant.Asset) gen.MediaAsset {
	return gen.MediaAsset{
		Id:          gen.Uuid(asset.ID),
		Url:         a.media.PublicURL(asset),
		ContentType: asset.ContentType,
		Size:        asset.Size,
		Width:       optionalInt(asset.Width),
		Height:      optionalInt(asset.Height),
		CreatedAt:   asset.CreatedAt,
	}
}

func (a *MerchantMediaAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *merchant.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeProblem(w, r, http.StatusUnprocessableEntity, invalid.Code, nil,
			map[string][]string{invalid.Field: {invalid.Code}})
	case errors.Is(err, merchant.ErrNoMerchant), errors.Is(err, merchant.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
	case errors.Is(err, merchant.ErrMediaNotFound), errors.Is(err, merchant.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "media_not_found", nil, nil)
	case errors.Is(err, merchant.ErrMediaInUse):
		writeProblem(w, r, http.StatusConflict, "media_in_use", nil, nil)
	case errors.Is(err, merchant.ErrMediaNotReady):
		writeProblem(w, r, http.StatusConflict, "media_already_stored", nil, nil)
	case errors.Is(err, merchant.ErrMediaFormat):
		writeProblem(w, r, http.StatusUnprocessableEntity, "media_format_mismatch", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "media request failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", strconv.Quote(err.Error())),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

// optionalInt returns nil for an unknown size, so clients never mistake 0 for a real dimension.
func optionalInt(value int) *int {
	if value <= 0 {
		return nil
	}
	return &value
}
