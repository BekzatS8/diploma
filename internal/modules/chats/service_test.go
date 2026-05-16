package chats

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
)

type mockRepo struct {
	participant bool
	createRes   CreateMessageResult
	createErr   error
}

func (m *mockRepo) EnsureChatForSelectionTx(ctx context.Context, tx pgx.Tx, orderID, clientID, executorID string) (string, error) {
	return "", nil
}
func (m *mockRepo) EnsureDirectChat(ctx context.Context, userID, peerID string) (ChatDetail, error) {
	return ChatDetail{}, nil
}
func (m *mockRepo) ListMyChats(ctx context.Context, q ListChatsQuery) ([]ChatSummary, int64, error) {
	return nil, 0, nil
}
func (m *mockRepo) GetMyChatByID(ctx context.Context, chatID, userID string) (ChatDetail, error) {
	return ChatDetail{}, nil
}
func (m *mockRepo) IsParticipant(ctx context.Context, chatID, userID string) (bool, error) {
	return m.participant, nil
}
func (m *mockRepo) ListMessages(ctx context.Context, chatID string, q ListMessagesQuery) ([]Message, int64, error) {
	return []Message{}, 0, nil
}
func (m *mockRepo) CreateMessage(ctx context.Context, chatID, senderUserID, text string, uploadIDs []string) (CreateMessageResult, error) {
	if m.createErr != nil {
		return CreateMessageResult{}, m.createErr
	}
	return m.createRes, nil
}
func (m *mockRepo) UpdateMessage(ctx context.Context, chatID, messageID, userID, text string) (Message, error) {
	return Message{}, nil
}
func (m *mockRepo) DeleteMessage(ctx context.Context, chatID, messageID, userID string) error {
	return nil
}
func (m *mockRepo) MarkRead(ctx context.Context, chatID, userID string) error { return nil }
func (m *mockRepo) ListAdminChats(ctx context.Context, page, pageSize int) ([]ChatSummary, int64, error) {
	return nil, 0, nil
}
func (m *mockRepo) GetChatByIDAdmin(ctx context.Context, chatID string) (ChatDetail, error) {
	return ChatDetail{}, nil
}
func (m *mockRepo) ListMessagesAdmin(ctx context.Context, chatID string, q ListMessagesQuery) ([]Message, int64, error) {
	return nil, 0, nil
}

func TestSendMessageMy_RejectsAdminAndEmpty(t *testing.T) {
	svc := NewService(&mockRepo{}, nil, nil)

	if _, err := svc.SendMessageMy(context.Background(), "chat", "u1", "admin", "hello", nil); err != ErrForbidden {
		t.Fatalf("expected ErrForbidden for admin, got %v", err)
	}
	if _, err := svc.SendMessageMy(context.Background(), "chat", "u1", "executor", "   ", nil); err != ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput for empty message, got %v", err)
	}
}

func TestListMessagesMy_OnlyParticipant(t *testing.T) {
	svc := NewService(&mockRepo{participant: false}, nil, nil)
	_, _, err := svc.ListMessagesMy(context.Background(), "chat", "user", "executor", ListMessagesQuery{Page: 1, PageSize: 20})
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound for non participant, got %v", err)
	}
}
