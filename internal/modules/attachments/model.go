package attachments

import (
	"encoding/json"
	"time"
)

type TargetType string

const (
	TargetProfileDocument    TargetType = "profile_document"
	TargetOrderAttachment    TargetType = "order_attachment"
	TargetResponseAttachment TargetType = "response_attachment"
	TargetReviewAttachment   TargetType = "review_attachment"
	TargetChatAttachment     TargetType = "chat_attachment"
	TargetCourseMaterial     TargetType = "course_material"
)

type Metadata map[string]any

type Attachment struct {
	ID         string
	UploadID   string
	TargetType TargetType
	TargetID   string
	SortOrder  int
	Metadata   Metadata
	CreatedAt  time.Time
}

func (m Metadata) jsonBytes() ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}
