package merchant

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

// Document kinds required for review (FR-01.2).
const (
	DocumentCertificate = "certificate"
	DocumentBankLetter  = "bank_letter"
	// DocumentPriceProof is optional: a price list or menu that supports the deal prices.
	DocumentPriceProof = "price_proof"
)

// MaxDocumentBytes caps one upload. Scans of a certificate or a bank letter fit well below it.
const MaxDocumentBytes = 10 << 20

// Document errors. The HTTP layer maps them to problem codes.
var (
	ErrDocumentKind      = &InvalidInputError{Field: "kind", Code: "invalid_document_kind"}
	ErrDocumentType      = &InvalidInputError{Field: "content_type", Code: "unsupported_document_type"}
	ErrDocumentTooLarge  = &InvalidInputError{Field: "body", Code: "document_too_large"}
	ErrDocumentEmpty     = &InvalidInputError{Field: "body", Code: "document_empty"}
	ErrDocumentsLocked   = errors.New("merchant: documents can only change while the business is a draft or rejected")
	ErrDocumentNotFound  = errors.New("merchant: document not found")
	ErrDocumentsRequired = errors.New("merchant: certificate and bank letter must be uploaded before submitting")
)

// documentExtensions maps accepted content types to the stored file extension.
var documentExtensions = map[string]string{
	"application/pdf": "pdf",
	"image/jpeg":      "jpg",
	"image/png":       "png",
}

// Document is the metadata of one stored file. The bytes stay in the private store.
type Document struct {
	Kind        string
	ContentType string
	SizeBytes   int64
	UploadedAt  time.Time
	StorageKey  string
}

// DocumentStore keeps the bytes. It is private: nothing reads it except this package's services.
type DocumentStore interface {
	Put(ctx context.Context, key string, body io.Reader) error
	Open(key string) (io.ReadCloser, error)
	Delete(key string) error
}

// DocumentRepository persists document metadata. SaveDocument returns the key of the file it replaced.
type DocumentRepository interface {
	OwnedBusiness(ctx context.Context, ownerID uuid.UUID) (uuid.UUID, string, error)
	SaveDocument(ctx context.Context, merchantID, uploaderID uuid.UUID, doc Document) (previousKey string, err error)
	Documents(ctx context.Context, merchantID uuid.UUID) ([]Document, error)
	Document(ctx context.Context, merchantID uuid.UUID, kind string) (Document, error)
}

// DocumentService stores business documents for review.
type DocumentService struct {
	repo  DocumentRepository
	files DocumentStore
}

// NewDocumentService binds the document rules to their repository and private store.
func NewDocumentService(repo DocumentRepository, files DocumentStore) *DocumentService {
	return &DocumentService{repo: repo, files: files}
}

// Upload stores one document for the owner's business. Documents can change only before review
// (draft) or after a rejection, so reviewers always read what was submitted.
func (s *DocumentService) Upload(ctx context.Context, ownerID uuid.UUID, kind, contentType string, body io.Reader) (Document, error) {
	if kind != DocumentCertificate && kind != DocumentBankLetter && kind != DocumentPriceProof {
		return Document{}, ErrDocumentKind
	}
	ext, ok := documentExtensions[contentType]
	if !ok {
		return Document{}, ErrDocumentType
	}
	data, err := io.ReadAll(io.LimitReader(body, MaxDocumentBytes+1))
	if err != nil {
		return Document{}, fmt.Errorf("read document: %w", err)
	}
	if len(data) == 0 {
		return Document{}, ErrDocumentEmpty
	}
	if len(data) > MaxDocumentBytes {
		return Document{}, ErrDocumentTooLarge
	}

	merchantID, status, err := s.repo.OwnedBusiness(ctx, ownerID)
	if err != nil {
		return Document{}, err
	}
	// Identity documents are locked once a business is submitted. A price proof backs deal pricing, so
	// an approved business may still replace it.
	editable := status == "draft" || status == "rejected" || (kind == DocumentPriceProof && status == "approved")
	if !editable {
		return Document{}, ErrDocumentsLocked
	}

	key, err := newDocumentKey(ext)
	if err != nil {
		return Document{}, err
	}
	if err := s.files.Put(ctx, key, bytes.NewReader(data)); err != nil {
		return Document{}, fmt.Errorf("store document: %w", err)
	}
	doc := Document{Kind: kind, ContentType: contentType, SizeBytes: int64(len(data)), UploadedAt: time.Now(), StorageKey: key}
	previous, err := s.repo.SaveDocument(ctx, merchantID, ownerID, doc)
	if err != nil {
		_ = s.files.Delete(key)
		return Document{}, err
	}
	if previous != "" {
		_ = s.files.Delete(previous)
	}
	return doc, nil
}

// List returns the owner's uploaded documents, without the storage keys.
func (s *DocumentService) List(ctx context.Context, ownerID uuid.UUID) ([]Document, error) {
	merchantID, _, err := s.repo.OwnedBusiness(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	return s.repo.Documents(ctx, merchantID)
}

// Open returns a document for a platform reviewer. Owners cannot read documents this way.
func (s *DocumentService) Open(ctx context.Context, actor ReviewActor, merchantID uuid.UUID, kind string) (Document, io.ReadCloser, error) {
	if !reviewerRoles[actor.Role] {
		return Document{}, nil, ErrForbidden
	}
	doc, err := s.repo.Document(ctx, merchantID, kind)
	if err != nil {
		return Document{}, nil, err
	}
	file, err := s.files.Open(doc.StorageKey)
	if err != nil {
		return Document{}, nil, ErrDocumentNotFound
	}
	return doc, file, nil
}

func newDocumentKey(ext string) (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate document key: %w", err)
	}
	id := uuid.Must(uuid.FromBytes(raw)).String()
	return id + "." + ext, nil
}
