package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

// privateKeyPattern accepts only generated names: a UUID and a document extension.
var privateKeyPattern = regexp.MustCompile(`^[a-f0-9-]{36}\.(pdf|jpg|png)$`)

// Private stores files that must never be served publicly, such as business documents. Files are
// read only through the API, by reviewers. Nothing here builds a public URL.
type Private struct {
	root string
}

// NewPrivate creates the directory if needed and returns the store.
func NewPrivate(root string) (*Private, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create private storage: %w", err)
	}
	return &Private{root: root}, nil
}

// Put writes the object atomically: a temporary file is renamed only once it is complete.
func (p *Private) Put(_ context.Context, key string, body io.Reader) error {
	path, err := p.path(key)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(p.root, "upload-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmp.Name()) //nolint:errcheck
	if _, err := io.Copy(tmp, body); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write document: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close document: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("store document: %w", err)
	}
	return nil
}

// Open returns the object for reading. The caller must close it.
func (p *Private) Open(key string) (io.ReadCloser, error) {
	path, err := p.path(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("open document: %w", err)
	}
	return file, nil
}

// Delete removes the object. A missing object is not an error.
func (p *Private) Delete(key string) error {
	path, err := p.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete document: %w", err)
	}
	return nil
}

func (p *Private) path(key string) (string, error) {
	if !privateKeyPattern.MatchString(key) {
		return "", fmt.Errorf("storage: invalid private key %q", key)
	}
	return filepath.Join(p.root, key), nil
}
