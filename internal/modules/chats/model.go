package chats

import "time"

type ChatParticipant struct {
	UserID     string     `json:"user_id"`
	JoinedAt   time.Time  `json:"joined_at"`
	LastReadAt *time.Time `json:"last_read_at,omitempty"`
}

type ChatSummary struct {
	ChatID             string            `json:"chat_id"`
	OrderID            string            `json:"order_id"`
	Participants       []ChatParticipant `json:"participants"`
	LastMessagePreview *string           `json:"last_message_preview,omitempty"`
	LastMessageAt      *time.Time        `json:"last_message_at,omitempty"`
	UnreadCount        int64             `json:"unread_count"`
	HasUnread          bool              `json:"has_unread"`
}

type ChatDetail struct {
	ChatID             string            `json:"chat_id"`
	OrderID            string            `json:"order_id"`
	OrderStatus        string            `json:"order_status"`
	ClientID           string            `json:"client_id"`
	SelectedExecutorID *string           `json:"selected_executor_id,omitempty"`
	Participants       []ChatParticipant `json:"participants"`
	LastMessageAt      *time.Time        `json:"last_message_at,omitempty"`
}

type Message struct {
	ID           string    `json:"id"`
	ChatID       string    `json:"chat_id"`
	SenderUserID *string   `json:"sender_user_id,omitempty"`
	SenderType   string    `json:"sender_type"`
	Text         string    `json:"text"`
	CreatedAt    time.Time `json:"created_at"`
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
