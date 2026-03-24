package chats

import (
	"context"
	"errors"
	"strings"

	notifications "buhpro/internal/modules/notifications"

	"github.com/jackc/pgx/v5"
)

var (
	ErrForbidden    = errors.New("forbidden")
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
)

type Service struct {
	repo     chatRepository
	notifier *notifications.Service
}

type chatRepository interface {
	EnsureChatForSelectionTx(ctx context.Context, tx pgx.Tx, orderID, clientID, executorID string) (string, error)
	ListMyChats(ctx context.Context, q ListChatsQuery) ([]ChatSummary, int64, error)
	GetMyChatByID(ctx context.Context, chatID, userID string) (ChatDetail, error)
	IsParticipant(ctx context.Context, chatID, userID string) (bool, error)
	ListMessages(ctx context.Context, chatID string, q ListMessagesQuery) ([]Message, int64, error)
	CreateMessage(ctx context.Context, chatID, senderUserID, text string) (CreateMessageResult, error)
	MarkRead(ctx context.Context, chatID, userID string) error
	ListAdminChats(ctx context.Context, page, pageSize int) ([]ChatSummary, int64, error)
	GetChatByIDAdmin(ctx context.Context, chatID string) (ChatDetail, error)
	ListMessagesAdmin(ctx context.Context, chatID string, q ListMessagesQuery) ([]Message, int64, error)
}

func NewService(repo chatRepository, notifier *notifications.Service) *Service {
	return &Service{repo: repo, notifier: notifier}
}

func (s *Service) EnsureChatForSelectionTx(ctx context.Context, tx pgx.Tx, orderID, clientID, executorID string) error {
	_, err := s.repo.EnsureChatForSelectionTx(ctx, tx, orderID, clientID, executorID)
	return err
}

func (s *Service) ListMyChats(ctx context.Context, userID, role string, q ListChatsQuery) ([]ChatSummary, int64, error) {
	if role == "" {
		return nil, 0, ErrForbidden
	}
	normalizeChatsQuery(&q)
	q.UserID = userID
	return s.repo.ListMyChats(ctx, q)
}

func (s *Service) GetMyChatByID(ctx context.Context, chatID, userID, role string) (ChatDetail, error) {
	if role == "" {
		return ChatDetail{}, ErrForbidden
	}
	item, err := s.repo.GetMyChatByID(ctx, chatID, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChatDetail{}, ErrNotFound
		}
		return ChatDetail{}, err
	}
	return item, nil
}

func (s *Service) ListMessagesMy(ctx context.Context, chatID, userID, role string, q ListMessagesQuery) ([]Message, int64, error) {
	if role == "" {
		return nil, 0, ErrForbidden
	}
	normalizeMessagesQuery(&q)
	ok, err := s.repo.IsParticipant(ctx, chatID, userID)
	if err != nil {
		return nil, 0, err
	}
	if !ok {
		return nil, 0, ErrNotFound
	}
	return s.repo.ListMessages(ctx, chatID, q)
}

func (s *Service) SendMessageMy(ctx context.Context, chatID, userID, role string, text string) (Message, error) {
	if role == "" || role == "admin" {
		return Message{}, ErrForbidden
	}
	body := strings.TrimSpace(text)
	if body == "" {
		return Message{}, ErrInvalidInput
	}
	result, err := s.repo.CreateMessage(ctx, chatID, userID, body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Message{}, ErrNotFound
		}
		return Message{}, err
	}
	if s.notifier != nil && result.ReceiverSet && result.ReceiverID != "" {
		_, _ = s.notifier.EmitInApp(ctx, result.ReceiverID, notifications.TypeChatMessageReceived, map[string]any{
			"chat_id":    chatID,
			"order_id":   result.OrderID,
			"message_id": result.Message.ID,
		})
	}
	return result.Message, nil
}

func (s *Service) MarkReadMy(ctx context.Context, chatID, userID, role string) error {
	if role == "" {
		return ErrForbidden
	}
	if err := s.repo.MarkRead(ctx, chatID, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *Service) ListAdminChats(ctx context.Context, role string, page, pageSize int) ([]ChatSummary, int64, error) {
	if role != "admin" {
		return nil, 0, ErrForbidden
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListAdminChats(ctx, page, pageSize)
}

func (s *Service) GetAdminChatByID(ctx context.Context, role, chatID string) (ChatDetail, error) {
	if role != "admin" {
		return ChatDetail{}, ErrForbidden
	}
	item, err := s.repo.GetChatByIDAdmin(ctx, chatID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ChatDetail{}, ErrNotFound
		}
		return ChatDetail{}, err
	}
	return item, nil
}

func (s *Service) ListAdminMessages(ctx context.Context, role, chatID string, q ListMessagesQuery) ([]Message, int64, error) {
	if role != "admin" {
		return nil, 0, ErrForbidden
	}
	normalizeMessagesQuery(&q)
	items, total, err := s.repo.ListMessagesAdmin(ctx, chatID, q)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, ErrNotFound
		}
		return nil, 0, err
	}
	return items, total, nil
}

func normalizeChatsQuery(q *ListChatsQuery) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
}

func normalizeMessagesQuery(q *ListMessagesQuery) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
}
