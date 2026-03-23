package storage

import (
	"context"
	"fmt"
	"io"
)

type ObjectInfo struct {
	Key string
	URL string
}

type Provider interface {
	Upload(ctx context.Context, key string, reader io.Reader, contentType string) (ObjectInfo, error)
	Delete(ctx context.Context, key string) error
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
