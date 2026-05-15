package attachments

import "time"

type AttachRequest struct {
	UploadIDs  []string `json:"upload_ids" binding:"required"`
	TargetType string   `json:"target_type" binding:"required"`
	TargetID   string   `json:"target_id" binding:"required"`
	Metadata   Metadata `json:"metadata"`
}

type ReorderRequest struct {
	IDs []string `json:"ids" binding:"required"`
}

type AttachmentView struct {
	ID           string     `json:"id"`
	UploadID     string     `json:"upload_id"`
	TargetType   TargetType `json:"target_type"`
	TargetID     string     `json:"target_id"`
	SortOrder    int        `json:"sort_order"`
	Metadata     Metadata   `json:"metadata"`
	CreatedAt    time.Time  `json:"created_at"`
	URL          string     `json:"url"`
	OriginalName string     `json:"original_name"`
	MimeType     string     `json:"mime_type"`
	SizeBytes    int64      `json:"size_bytes"`
}

type AttachmentListResponse struct {
	Items []AttachmentView `json:"items"`
}
