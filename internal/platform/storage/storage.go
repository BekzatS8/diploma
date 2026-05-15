package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ObjectInfo struct {
	Key string
	URL string
}

type Provider interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) (ObjectInfo, error)
	Delete(ctx context.Context, key string) error
	URL(key string) string
}

type MockStorage struct{}

func NewMock() Provider {
	return &MockStorage{}
}

func (m *MockStorage) Upload(_ context.Context, key string, _ io.Reader, _ string) (ObjectInfo, error) {
	return ObjectInfo{Key: key, URL: fmt.Sprintf("https://mock-storage.local/%s", key)}, nil
}

func (m *MockStorage) Delete(_ context.Context, _ string) error {
	return nil
}

func (m *MockStorage) URL(key string) string {
	return fmt.Sprintf("https://mock-storage.local/%s", key)
}

type LocalStorage struct {
	basePath string
	baseURL  string
}

func NewLocal(basePath, baseURL string) (*LocalStorage, error) {
	if strings.TrimSpace(basePath) == "" {
		basePath = "uploads"
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "/uploads"
	}
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("create storage directory: %w", err)
	}
	return &LocalStorage{basePath: basePath, baseURL: strings.TrimRight(baseURL, "/")}, nil
}

func (s *LocalStorage) Upload(_ context.Context, key string, reader io.Reader, contentType string) (ObjectInfo, error) {
	fullPath, err := s.fullPath(key)
	if err != nil {
		return ObjectInfo{}, err
	}
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return ObjectInfo{}, fmt.Errorf("create storage subdirectory: %w", err)
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("create storage object: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		_ = os.Remove(fullPath)
		return ObjectInfo{}, fmt.Errorf("write storage object: %w", err)
	}

	return ObjectInfo{Key: key, URL: s.URL(key)}, nil
}

func (s *LocalStorage) Delete(_ context.Context, key string) error {
	fullPath, err := s.fullPath(key)
	if err != nil {
		return err
	}
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete storage object: %w", err)
	}
	return nil
}

func (s *LocalStorage) URL(key string) string {
	return s.baseURL + "/" + strings.TrimLeft(filepath.ToSlash(key), "/")
}

func (s *LocalStorage) fullPath(key string) (string, error) {
	cleanKey := filepath.Clean(filepath.FromSlash(strings.TrimLeft(key, "/\\")))
	if cleanKey == "." || strings.HasPrefix(cleanKey, "..") || filepath.IsAbs(cleanKey) {
		return "", fmt.Errorf("invalid storage key")
	}
	return filepath.Join(s.basePath, cleanKey), nil
}
