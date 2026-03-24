package notifications

import (
	"encoding/json"
	"time"
)

type Notification struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	Type      string          `json:"type"`
	Channel   string          `json:"channel"`
	Status    string          `json:"status"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
	SentAt    *time.Time      `json:"sent_at,omitempty"`
	ReadAt    *time.Time      `json:"read_at,omitempty"`
}

const (
	ChannelInApp = "in_app"

	StatusPending = "pending"
	StatusSent    = "sent"
	StatusFailed  = "failed"
	StatusRead    = "read"

	TypeOrderPublished      = "order_published"
	TypeResponseSubmitted   = "response_submitted"
	TypeResponseSelected    = "response_selected"
	TypeOrderCompleted      = "order_completed"
	TypeReviewCreated       = "review_created"
	TypeSanctionCreated     = "sanction_created"
	TypeCourseAssigned      = "course_assigned"
	TypeCourseCompleted     = "course_completed"
	TypeChatMessageReceived = "chat_message_received"
)
