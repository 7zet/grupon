package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/dealo/dealo/services/api/internal/domain/identity"
	"github.com/dealo/dealo/services/api/internal/domain/merchant"
	"github.com/dealo/dealo/services/api/internal/http/gen"
)

// MerchantDocumentAPI serves business documents: owners upload, reviewers download (FR-01.2). Files are
// never served as public media.
type MerchantDocumentAPI struct {
	service *merchant.DocumentService
	review  *merchant.ReviewService
	auth    *AuthAPI
	logger  *slog.Logger
}

// NewMerchantDocumentAPI binds the document rules to the HTTP contract.
func NewMerchantDocumentAPI(service *merchant.DocumentService, review *merchant.ReviewService, auth *AuthAPI, logger *slog.Logger) *MerchantDocumentAPI {
	return &MerchantDocumentAPI{service: service, review: review, auth: auth, logger: logger}
}

func (a *MerchantDocumentAPI) ListMerchantDocuments(w http.ResponseWriter, r *http.Request) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	docs, err := a.service.List(r.Context(), account.ID)
	if err != nil {
		a.fail(w, r, err)
		return
	}
	items := make([]gen.MerchantDocument, 0, len(docs))
	for _, doc := range docs {
		items = append(items, mapMerchantDocument(doc))
	}
	writeJSON(w, http.StatusOK, gen.MerchantDocumentList{Items: items})
}

func (a *MerchantDocumentAPI) UploadMerchantDocument(w http.ResponseWriter, r *http.Request, kind gen.UploadMerchantDocumentParamsKind) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	// One byte above the limit lets the domain tell "too large" from a short body.
	body := http.MaxBytesReader(w, r.Body, merchant.MaxDocumentBytes+1)
	defer body.Close()
	contentType := r.Header.Get("Content-Type")
	if _, err := a.service.Upload(r.Context(), account.ID, string(kind), contentType, body); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeProblem(w, r, http.StatusUnprocessableEntity, "document_too_large", nil, nil)
			return
		}
		a.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *MerchantDocumentAPI) DownloadMerchantDocument(w http.ResponseWriter, r *http.Request, id gen.Id, kind gen.DownloadMerchantDocumentParamsKind) {
	account, ok := a.auth.authenticate(w, r)
	if !ok {
		return
	}
	actor, ok := reviewActor(account)
	if !ok {
		writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
		return
	}
	doc, file, err := a.service.Open(r.Context(), actor, uuid.UUID(id), string(kind))
	if err != nil {
		a.fail(w, r, err)
		return
	}
	defer file.Close()
	w.Header().Set("Content-Type", doc.ContentType)
	w.Header().Set("Content-Disposition", "attachment")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if _, err := io.Copy(w, file); err != nil {
		a.logger.WarnContext(r.Context(), "document download interrupted", slog.String("error", err.Error()))
	}
}

// reviewActor picks the highest review role an account holds, the same order the review API uses.
func reviewActor(account identity.Account) (merchant.ReviewActor, bool) {
	for _, role := range []string{"superadmin", "admin", "moderator"} {
		if account.HasRole(identity.Role(role)) {
			return merchant.ReviewActor{UserID: account.ID, Role: role}, true
		}
	}
	return merchant.ReviewActor{}, false
}

func (a *MerchantDocumentAPI) fail(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *merchant.InvalidInputError
	switch {
	case errors.As(err, &invalid):
		writeProblem(w, r, http.StatusUnprocessableEntity, invalid.Code, nil,
			map[string][]string{invalid.Field: {invalid.Code}})
	case errors.Is(err, merchant.ErrNoBusiness), errors.Is(err, merchant.ErrDocumentNotFound):
		writeProblem(w, r, http.StatusNotFound, "not_found", nil, nil)
	case errors.Is(err, merchant.ErrDocumentsLocked):
		writeProblem(w, r, http.StatusConflict, "documents_locked", nil, nil)
	case errors.Is(err, merchant.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", nil, nil)
	default:
		a.logger.ErrorContext(r.Context(), "merchant document failed",
			slog.String("request_id", middleware.GetReqID(r.Context())),
			slog.String("error", err.Error()),
		)
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", nil, nil)
	}
}

func mapMerchantDocument(doc merchant.Document) gen.MerchantDocument {
	return gen.MerchantDocument{
		Kind:        gen.MerchantDocumentKind(doc.Kind),
		ContentType: gen.MerchantDocumentContentType(doc.ContentType),
		SizeBytes:   int(doc.SizeBytes),
		UploadedAt:  doc.UploadedAt.Round(time.Second),
	}
}
