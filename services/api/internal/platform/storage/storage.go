// Package storage stores uploaded media behind a small port. The local adapter serves development;
// an S3-compatible adapter (MinIO in development, object storage in production) implements the same
// port and must be tested before launch.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ErrNotFound reports a missing object.
var ErrNotFound = errors.New("storage: object not found")

// Store saves and reads immutable objects by key.
type Store interface {
	Put(ctx context.Context, key string, body io.Reader) error
	Delete(ctx context.Context, key string) error
	PublicURL(key string) string
}

var keyPattern = regexp.MustCompile(`^[a-z0-9]{2}/[a-f0-9-]{36}\.(jpg|png|webp)$`)

// Local stores objects under a directory and serves them from a public base path.
type Local struct {
	root     string
	basePath string
}

// NewLocal creates a local store. basePath is the URL prefix the API serves files under.
func NewLocal(root, basePath string) (*Local, error) {
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}
	return &Local{root: root, basePath: strings.TrimRight(basePath, "/")}, nil
}

// Put writes an object atomically: data lands under a temporary name and is renamed into place.
func (s *Local) Put(_ context.Context, key string, body io.Reader) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create object directory: %w", err)
	}
	temporary := path + ".part"
	file, err := os.OpenFile(temporary, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return fmt.Errorf("open object: %w", err)
	}
	if _, err := io.Copy(file, body); err != nil {
		_ = file.Close()
		_ = os.Remove(temporary)
		return fmt.Errorf("write object: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(temporary)
		return fmt.Errorf("close object: %w", err)
	}
	return os.Rename(temporary, path)
}

// Delete removes an object. A missing object is not an error.
func (s *Local) Delete(_ context.Context, key string) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

// PublicURL is the URL a browser uses to fetch the object.
func (s *Local) PublicURL(key string) string {
	return s.basePath + "/" + key
}

// Root exposes the directory for the static file handler.
func (s *Local) Root() string {
	return s.root
}

func (s *Local) path(key string) (string, error) {
	if !keyPattern.MatchString(key) {
		return "", fmt.Errorf("storage: invalid key %q", key)
	}
	return filepath.Join(s.root, filepath.FromSlash(key)), nil
}
