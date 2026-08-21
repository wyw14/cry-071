package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalStore struct{ root string }

func NewLocalStore(root string) (*LocalStore, error) {
	root = filepath.Clean(root)
	if root == "." || strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("attachment root is invalid")
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return nil, fmt.Errorf("create attachment root: %w", err)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve attachment root: %w", err)
	}
	return &LocalStore{root: absolute}, nil
}

func (s *LocalStore) Put(ctx context.Context, key string, reader io.Reader, expectedSize int64) (string, string, error) {
	path, err := s.path(key)
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", "", err
	}
	object, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o640)
	if err != nil {
		return "", "", err
	}
	closed := false
	defer func() {
		if !closed {
			_ = object.Close()
		}
	}()

	hasher := sha256.New()
	limited := io.LimitReader(reader, expectedSize+1)
	written, copyErr := io.Copy(io.MultiWriter(object, hasher), limited)
	if copyErr != nil {
		return "", "", copyErr
	}
	if written != expectedSize {
		return "", "", fmt.Errorf("attachment size mismatch: expected %d, got %d", expectedSize, written)
	}
	if err := object.Sync(); err != nil {
		return "", "", err
	}
	if err := object.Close(); err != nil {
		return "", "", err
	}
	closed = true
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	return key, hex.EncodeToString(hasher.Sum(nil)), nil
}

func (s *LocalStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	path, err := s.path(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (s *LocalStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *LocalStore) path(key string) (string, error) {
	key = filepath.Clean(filepath.FromSlash(strings.TrimSpace(key)))
	if key == "." || filepath.IsAbs(key) || strings.HasPrefix(key, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("unsafe storage key")
	}
	path := filepath.Join(s.root, key)
	relative, err := filepath.Rel(s.root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("storage key escapes root")
	}
	return path, nil
}
