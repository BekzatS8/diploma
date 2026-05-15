package uploads

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"buhpro/internal/platform/storage"

	"github.com/google/uuid"
)

const MaxUploadSize = 50 * 1024 * 1024

var allowedMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/gif":  true,

	"video/mp4":       true,
	"video/quicktime": true,

	"audio/mpeg":  true,
	"audio/wav":   true,
	"audio/x-wav": true,

	"application/pdf":    true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
	"application/zip": true,
	"text/plain":      true,
}

type Service struct {
	repo    *Repository
	storage storage.Provider
}

func NewService(repo *Repository, storage storage.Provider) *Service {
	return &Service{repo: repo, storage: storage}
}

func (s *Service) Upload(ctx context.Context, authorID, filename string, reader io.Reader) (Upload, error) {
	if strings.TrimSpace(authorID) == "" || reader == nil {
		return Upload{}, ErrInvalidInput
	}

	buf := &bytes.Buffer{}
	size, err := io.Copy(buf, io.LimitReader(reader, MaxUploadSize+1))
	if err != nil {
		return Upload{}, fmt.Errorf("read file: %w", err)
	}
	if size == 0 {
		return Upload{}, ErrInvalidInput
	}
	if size > MaxUploadSize {
		return Upload{}, ErrFileTooLarge
	}

	mimeType := http.DetectContentType(buf.Bytes())
	mimeType, _, _ = mime.ParseMediaType(mimeType)
	if !allowedMimeTypes[mimeType] {
		return Upload{}, ErrInvalidMime
	}

	fileID := uuid.NewString()
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		if exts, _ := mime.ExtensionsByType(mimeType); len(exts) > 0 {
			ext = exts[0]
		}
	}
	filePath := fmt.Sprintf("%s/%s%s", authorID, fileID, ext)

	if _, err := s.storage.Upload(ctx, filePath, bytes.NewReader(buf.Bytes()), mimeType); err != nil {
		return Upload{}, fmt.Errorf("save file: %w", err)
	}

	item := Upload{
		ID:           fileID,
		AuthorID:     authorID,
		FilePath:     filePath,
		OriginalName: safeFilename(filename),
		MimeType:     mimeType,
		SizeBytes:    size,
		CreatedAt:    time.Now(),
	}
	if err := s.repo.Create(ctx, item); err != nil {
		_ = s.storage.Delete(ctx, filePath)
		return Upload{}, fmt.Errorf("create upload record: %w", err)
	}

	return item, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (Upload, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListByAuthor(ctx context.Context, userID, role string) ([]Upload, error) {
	if role == "" {
		return nil, ErrForbidden
	}
	return s.repo.ListByAuthor(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, id, userID, role string) error {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if role != "admin" && item.AuthorID != userID {
		return ErrForbidden
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.storage.Delete(ctx, item.FilePath)
	return nil
}

func (s *Service) URL(item Upload) string {
	return s.storage.URL(item.FilePath)
}

func (s *Service) ToView(item Upload, includeAuthor bool) UploadView {
	view := UploadView{
		ID:           item.ID,
		FilePath:     item.FilePath,
		URL:          s.URL(item),
		OriginalName: item.OriginalName,
		MimeType:     item.MimeType,
		SizeBytes:    item.SizeBytes,
		CreatedAt:    item.CreatedAt,
	}
	if includeAuthor {
		view.AuthorID = item.AuthorID
	}
	return view
}

func safeFilename(filename string) string {
	name := strings.ReplaceAll(filename, "\\", "/")
	name = filepath.Base(name)
	if strings.TrimSpace(name) == "" || name == "." {
		return "upload"
	}
	return name
}
