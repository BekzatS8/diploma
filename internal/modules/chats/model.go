package chats

import "time"

type ChatParticipant struct {
	UserID     string     `json:"user_id"`
	JoinedAt   time.Time  `json:"joined_at"`
	LastReadAt *time.Time `json:"last_read_at,omitempty"`
}

type ChatSummary struct {
	ChatID             string            `json:"chat_id"`
	ChatType           string            `json:"chat_type"`
	OrderID            *string           `json:"order_id,omitempty"`
	UserAID            *string           `json:"user_a_id,omitempty"`
	UserBID            *string           `json:"user_b_id,omitempty"`
	Participants       []ChatParticipant `json:"participants"`
	LastMessagePreview *string           `json:"last_message_preview,omitempty"`
	LastMessageAt      *time.Time        `json:"last_message_at,omitempty"`
	UnreadCount        int64             `json:"unread_count"`
	HasUnread          bool              `json:"has_unread"`
}

type ChatDetail struct {
	ChatID             string            `json:"chat_id"`
	ChatType           string            `json:"chat_type"`
	OrderID            *string           `json:"order_id,omitempty"`
	OrderStatus        *string           `json:"order_status,omitempty"`
	ClientID           *string           `json:"client_id,omitempty"`
	SelectedExecutorID *string           `json:"selected_executor_id,omitempty"`
	UserAID            *string           `json:"user_a_id,omitempty"`
	UserBID            *string           `json:"user_b_id,omitempty"`
	Participants       []ChatParticipant `json:"participants"`
	LastMessageAt      *time.Time        `json:"last_message_at,omitempty"`
}

type MessageAttachment struct {
	ID           string    `json:"id"`
	UploadID     string    `json:"upload_id"`
	FilePath     string    `json:"file_path"`
	URL          string    `json:"url,omitempty"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	CreatedAt    time.Time `json:"created_at"`
}

type Message struct {
	ID           string              `json:"id"`
	ChatID       string              `json:"chat_id"`
	SenderUserID *string             `json:"sender_user_id,omitempty"`
	SenderType   string              `json:"sender_type"`
	Text         string              `json:"text"`
	Attachments  []MessageAttachment `json:"attachments"`
	CreatedAt    time.Time           `json:"created_at"`
	EditedAt     *time.Time          `json:"edited_at,omitempty"`
	DeletedAt    *time.Time          `json:"deleted_at,omitempty"`
}

type MessagesListResponse struct {
	Items    []Message `json:"items"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	Total    int64     `json:"total"`
	Order    string    `json:"order"`
}

type ListChatsQuery struct {
	Page     int
	PageSize int
	UserID   string
}

type ListMessagesQuery struct {
	Page     int
	PageSize int
}
