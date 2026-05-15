package uploads

import "time"

type UploadView struct {
	ID           string    `json:"id"`
	AuthorID     string    `json:"author_id,omitempty"`
	FilePath     string    `json:"file_path"`
	URL          string    `json:"url"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	CreatedAt    time.Time `json:"created_at"`
}

type UploadListResponse struct {
	Items []UploadView `json:"items"`
}
