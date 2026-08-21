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
	temporary, err := os.CreateTemp(filepath.Dir(path), ".upload-*")
	if err != nil {
		return "", "", err
	}
	temporaryName := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryName)
		}
	}()
	hasher := sha256.New()
	written, err := copyContext(ctx, io.MultiWriter(temporary, hasher), reader, expectedSize+1)
	if err != nil {
		return "", "", err
	}
	if written != expectedSize {
		return "", "", fmt.Errorf("attachment size mismatch: expected %d, got %d", expectedSize, written)
	}
	if err := temporary.Sync(); err != nil {
		return "", "", err
	}
	if err := temporary.Close(); err != nil {
		return "", "", err
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return "", "", err
	}
	committed = true
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

func copyContext(ctx context.Context, writer io.Writer, reader io.Reader, limit int64) (int64, error) {
	buffer := make([]byte, 32*1024)
	limited := io.LimitReader(reader, limit)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		read, readErr := limited.Read(buffer)
		if read > 0 {
			written, writeErr := writer.Write(buffer[:read])
			total += int64(written)
			if writeErr != nil {
				return total, writeErr
			}
			if written != read {
				return total, io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
	}
}
