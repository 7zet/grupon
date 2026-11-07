package merchant

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type memDocRepo struct {
	status string
	saved  []Document
	docs   map[string]Document
}

func (m *memDocRepo) OwnedBusiness(context.Context, uuid.UUID) (uuid.UUID, string, error) {
	return uuid.New(), m.status, nil
}

func (m *memDocRepo) SaveDocument(_ context.Context, _, _ uuid.UUID, doc Document) (string, error) {
	m.saved = append(m.saved, doc)
	return "", nil
}

func (m *memDocRepo) Documents(context.Context, uuid.UUID) ([]Document, error) { return nil, nil }

func (m *memDocRepo) Document(_ context.Context, _ uuid.UUID, kind string) (Document, error) {
	if doc, ok := m.docs[kind]; ok {
		return doc, nil
	}
	return Document{}, ErrDocumentNotFound
}

type memFiles struct{ keys []string }

func (m *memFiles) Put(_ context.Context, key string, body io.Reader) error {
	m.keys = append(m.keys, key)
	_, _ = io.Copy(io.Discard, body)
	return nil
}

func (m *memFiles) Open(string) (io.ReadCloser, error) { return nil, errors.New("not used") }
func (m *memFiles) Delete(string) error                { return nil }

func TestUploadAcceptsOnlyKnownKindsTypesAndSizes(t *testing.T) {
	t.Parallel()
	service := NewDocumentService(&memDocRepo{status: "draft"}, &memFiles{})
	owner := uuid.New()
	ctx := context.Background()

	var invalid *InvalidInputError
	cases := map[string]struct {
		kind, contentType string
		body              string
		code              string
	}{
		"unknown kind": {"passport", "application/pdf", "x", "invalid_document_kind"},
		"unknown type": {DocumentCertificate, "text/html", "x", "unsupported_document_type"},
		"empty file":   {DocumentBankLetter, "image/png", "", "document_empty"},
		"too large":    {DocumentBankLetter, "application/pdf", strings.Repeat("a", MaxDocumentBytes+1), "document_too_large"},
	}
	for name, tc := range cases {
		_, err := service.Upload(ctx, owner, tc.kind, tc.contentType, strings.NewReader(tc.body))
		if !errors.As(err, &invalid) || invalid.Code != tc.code {
			t.Fatalf("%s: err = %v, want %s", name, err, tc.code)
		}
	}
}

func TestUploadIsLockedOnceUnderReview(t *testing.T) {
	t.Parallel()
	service := NewDocumentService(&memDocRepo{status: "submitted"}, &memFiles{})
	_, err := service.Upload(context.Background(), uuid.New(), DocumentCertificate, "application/pdf", bytes.NewReader([]byte("%PDF")))
	if !errors.Is(err, ErrDocumentsLocked) {
		t.Fatalf("upload while submitted: got %v, want ErrDocumentsLocked", err)
	}
}

func TestUploadStoresUnderAGeneratedKey(t *testing.T) {
	t.Parallel()
	repo := &memDocRepo{status: "rejected"}
	files := &memFiles{}
	service := NewDocumentService(repo, files)
	doc, err := service.Upload(context.Background(), uuid.New(), DocumentBankLetter, "image/jpeg", bytes.NewReader([]byte("jpeg-bytes")))
	if err != nil {
		t.Fatalf("upload after rejection: %v", err)
	}
	if !strings.HasSuffix(doc.StorageKey, ".jpg") || len(files.keys) != 1 || files.keys[0] != doc.StorageKey {
		t.Fatalf("storage key = %q, stored = %v", doc.StorageKey, files.keys)
	}
	if doc.SizeBytes != int64(len("jpeg-bytes")) || doc.UploadedAt.After(time.Now().Add(time.Minute)) {
		t.Fatalf("metadata = %+v", doc)
	}
}

func TestApprovedBusinessMayReplaceOnlyThePriceProof(t *testing.T) {
	t.Parallel()
	service := NewDocumentService(&memDocRepo{status: "approved"}, &memFiles{})
	ctx := context.Background()
	if _, err := service.Upload(ctx, uuid.New(), DocumentPriceProof, "application/pdf", bytes.NewReader([]byte("%PDF"))); err != nil {
		t.Fatalf("price proof while approved: %v", err)
	}
	if _, err := service.Upload(ctx, uuid.New(), DocumentCertificate, "application/pdf", bytes.NewReader([]byte("%PDF"))); !errors.Is(err, ErrDocumentsLocked) {
		t.Fatalf("certificate while approved: got %v, want ErrDocumentsLocked", err)
	}
}

func TestDemoPriceProofIsServedWithoutAStoredFile(t *testing.T) {
	t.Parallel()
	repo := &memDocRepo{status: "approved"}
	repo.docs = map[string]Document{DocumentPriceProof: {Kind: DocumentPriceProof, ContentType: "application/pdf", StorageKey: DemoDocumentKey("seed-merchant-01")}}
	service := NewDocumentService(repo, &memFiles{})
	_, body, err := service.Open(context.Background(), ReviewActor{UserID: uuid.New(), Role: "admin"}, uuid.New(), DocumentPriceProof)
	if err != nil {
		t.Fatalf("open demo proof: %v", err)
	}
	defer body.Close()
	data, _ := io.ReadAll(body)
	if !bytes.HasPrefix(data, []byte("%PDF-1.4")) || !bytes.Contains(data, []byte("DEMO")) {
		t.Fatalf("demo proof is not a marked PDF: %q", data[:20])
	}
}
