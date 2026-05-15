package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	commonauth "buhpro/internal/common/auth"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrInvalidRole          = errors.New("invalid role")
	ErrExecutorLeadRequired = errors.New("executor lead required")
)

type Service struct {
	repo *Repository
	jwt  *commonauth.JWTManager
}

func NewService(repo *Repository, jwt *commonauth.JWTManager) *Service {
	return &Service{repo: repo, jwt: jwt}
}

type RegisterInput struct {
	Email           string
	Password        string
	Role            string
	ProfileName     string
	Phone           string
	ClientType      string
	TaxNumber       string
	ContactName     string
	ContactPosition string
	Address         string
	About           string
	Website         string
	FirstName       string
	LastName        string
	MiddleName      string
	IIN             string
	City            string
	ExperienceLevel string
	Specializations []string
	Education       string
	WorkFormat      string
	HourlyRate      *float64
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (s *Service) BootstrapAdmin(ctx context.Context, email, password string) error {
	if err := ValidatePassword(password); err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	user := User{ID: uuid.NewString(), Email: strings.TrimSpace(strings.ToLower(email)), PasswordHash: hash, Role: "admin", IsActive: true, VerificationStatus: "verified"}
	if err := s.repo.CreateAdmin(ctx, user); err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			return nil
		}
		return err
	}
	return nil
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (User, TokenPair, error) {
	role := strings.ToLower(strings.TrimSpace(in.Role))
	if role != "client" && role != "executor" && role != "coach" {
		return User{}, TokenPair{}, ErrInvalidRole
	}
	if role == "executor" {
		return User{}, TokenPair{}, ErrExecutorLeadRequired
	}
	if err := ValidatePassword(in.Password); err != nil {
		return User{}, TokenPair{}, err
	}

	hash, err := HashPassword(in.Password)
	if err != nil {
		return User{}, TokenPair{}, err
	}

	user := User{
		ID:                 uuid.NewString(),
		Email:              strings.TrimSpace(strings.ToLower(in.Email)),
		PasswordHash:       hash,
		Role:               role,
		IsActive:           true,
		VerificationStatus: "verified",
	}
	if err := s.repo.CreateUserWithProfile(ctx, user, in); err != nil {
		return User{}, TokenPair{}, err
	}

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return User{}, TokenPair{}, err
	}
	return user, pair, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (User, TokenPair, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return User{}, TokenPair{}, ErrInvalidCredentials
	}
	if !user.IsActive {
		return User{}, TokenPair{}, ErrInvalidCredentials
	}
	if err := ComparePassword(user.PasswordHash, password); err != nil {
		return User{}, TokenPair{}, ErrInvalidCredentials
	}

	pair, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return User{}, TokenPair{}, err
	}
	return user, pair, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	claims, err := s.jwt.ParseRefreshToken(refreshToken)
	if err != nil {
		return TokenPair{}, ErrUnauthorized
	}
	stored, err := s.repo.GetRefreshToken(ctx, claims.ID)
	if err != nil {
		return TokenPair{}, ErrUnauthorized
	}
	if stored.TokenHash != hashToken(refreshToken) {
		return TokenPair{}, ErrUnauthorized
	}

	user, err := s.repo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		return TokenPair{}, ErrUnauthorized
	}
	if !user.IsActive {
		return TokenPair{}, ErrUnauthorized
	}

	newID := uuid.NewString()
	newRefresh, err := s.jwt.GenerateRefreshToken(user.ID, user.Role, newID)
	if err != nil {
		return TokenPair{}, err
	}
	if err := s.repo.RotateRefreshToken(ctx, claims.ID, RefreshToken{
		ID:        newID,
		UserID:    user.ID,
		TokenHash: hashToken(newRefresh),
		ExpiresAt: time.Now().Add(s.jwt.RefreshTTL()),
	}); err != nil {
		return TokenPair{}, err
	}

	access, err := s.jwt.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{AccessToken: access, RefreshToken: newRefresh}, nil
}

func (s *Service) Logout(ctx context.Context, userID, refreshToken string) error {
	claims, err := s.jwt.ParseRefreshToken(refreshToken)
	if err != nil {
		return ErrUnauthorized
	}
	if claims.UserID != userID {
		return ErrUnauthorized
	}
	return s.repo.RevokeRefreshToken(ctx, claims.ID, userID)
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	return s.repo.GetUserByID(ctx, userID)
}

func (s *Service) issueTokenPair(ctx context.Context, user User) (TokenPair, error) {
	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		return TokenPair{}, err
	}
	refreshID := uuid.NewString()
	refreshToken, err := s.jwt.GenerateRefreshToken(user.ID, user.Role, refreshID)
	if err != nil {
		return TokenPair{}, err
	}

	err = s.repo.CreateRefreshToken(ctx, RefreshToken{
		ID:        refreshID,
		UserID:    user.ID,
		TokenHash: hashToken(refreshToken),
		ExpiresAt: time.Now().Add(s.jwt.RefreshTTL()),
	})
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
