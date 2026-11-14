// Package storage is the file persistence boundary per spec §9: source
// files live outside Postgres behind a small interface so an S3 driver can
// replace the local disk implementation without touching callers.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Storage interface {
	Save(ctx context.Context, key string, data io.Reader) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

// Local keeps files under a root directory on disk.
type Local struct {
	root string
}

func NewLocal(root string) (*Local, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	return &Local{root: root}, nil
}

// safePath guards against traversal outside the storage root.
func (l *Local) safePath(key string) (string, error) {
	p := filepath.Join(l.root, filepath.Clean(key))
	if !strings.HasPrefix(p, filepath.Clean(l.root)+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid storage key %q", key)
	}
	return p, nil
}

func (l *Local) Save(_ context.Context, key string, data io.Reader) error {
	p, err := l.safePath(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, data)
	return err
}

func (l *Local) Open(_ context.Context, key string) (io.ReadCloser, error) {
	p, err := l.safePath(key)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

func (l *Local) Delete(_ context.Context, key string) error {
	p, err := l.safePath(key)
	if err != nil {
		return err
	}
	return os.Remove(p)
}

func (l *Local) Exists(_ context.Context, key string) (bool, error) {
	p, err := l.safePath(key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}
