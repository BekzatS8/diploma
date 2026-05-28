package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	commonauth "buhpro/internal/common/auth"
	"buhpro/internal/platform/mail"

	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrInvalidRole          = errors.New("invalid role")
	ErrExecutorLeadRequired = errors.New("executor lead required")
	ErrWrongCurrentPassword = errors.New("wrong current password")
	ErrSamePassword         = errors.New("new password must differ from current password")
	ErrInvalidResetToken    = errors.New("invalid or expired reset token")
)

const passwordResetTTL = time.Hour

type PasswordResetConfig struct {
	FrontendBaseURL string
	AppName         string
	IsProduction    bool
}

type Service struct {
	repo     *Repository
	jwt      *commonauth.JWTManager
	resetCfg PasswordResetConfig
	mailer   mail.Sender
}

func NewService(
	repo *Repository,
	jwt *commonauth.JWTManager,
	resetCfg PasswordResetConfig,
	mailer mail.Sender,
) *Service {
	if mailer == nil {
		mailer = mail.NoopSender{}
	}
	return &Service{repo: repo, jwt: jwt, resetCfg: resetCfg, mailer: mailer}
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
	isCoach, err := s.isCoachForToken(ctx, user)
	if err != nil {
		return TokenPair{}, err
	}
	newRefresh, err := s.jwt.GenerateRefreshToken(user.ID, user.Role, newID, isCoach)
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

	access, err := s.jwt.GenerateAccessToken(user.ID, user.Role, isCoach)
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

func (s *Service) RequestPasswordReset(ctx context.Context, email string) (ForgotPasswordResponse, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	resp := ForgotPasswordResponse{
		Message: "Если аккаунт с таким email существует, мы отправили инструкцию по восстановлению пароля",
	}

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return resp, nil
		}
		return resp, err
	}
	if !user.IsActive {
		return resp, nil
	}

	rawToken := uuid.NewString()
	expiresAt := time.Now().Add(passwordResetTTL)
	if err := s.repo.ReplacePasswordResetToken(ctx, user.ID, hashToken(rawToken), expiresAt); err != nil {
		return resp, err
	}

	resetURL := s.resetCfg.FrontendBaseURL + "/auth/reset-password?token=" + rawToken
	mailConfigured := s.mailer.Enabled()
	resp.MailConfigured = mailConfigured

	if mailConfigured {
		html, text := mail.PasswordResetEmail(s.resetCfg.AppName, resetURL, 1)
		subject := "Сброс пароля — " + s.resetCfg.AppName
		if err := s.mailer.Send(ctx, user.Email, subject, html, text); err != nil {
			slog.Error("password reset email failed", "email", user.Email, "error", err)
			if !s.resetCfg.IsProduction {
				resp.ResetURL = resetURL
			}
		} else {
			resp.EmailSent = true
			resp.Message = "Если аккаунт с таким email существует, на почту отправлена ссылка для сброса пароля"
		}
	} else if !s.resetCfg.IsProduction {
		resp.ResetURL = resetURL
	}

	return resp, nil
}

func (s *Service) ResetPasswordWithToken(ctx context.Context, token, newPassword string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrInvalidResetToken
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}

	userID, tokenID, err := s.repo.GetActivePasswordReset(ctx, hashToken(token))
	if err != nil {
		return ErrInvalidResetToken
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.UpdatePasswordHash(ctx, userID, hash); err != nil {
		return err
	}
	return s.repo.MarkPasswordResetUsed(ctx, tokenID)
}

func (s *Service) ChangePassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if err := ComparePassword(user.PasswordHash, currentPassword); err != nil {
		return ErrWrongCurrentPassword
	}
	if currentPassword == newPassword {
		return ErrSamePassword
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdatePasswordHash(ctx, userID, hash)
}

func (s *Service) IsCoachCapability(ctx context.Context, userID, role string) (bool, error) {
	if role == "coach" {
		return true, nil
	}
	if role != "executor" {
		return false, nil
	}
	return s.repo.HasCoachProfile(ctx, userID)
}

func (s *Service) isCoachForToken(ctx context.Context, user User) (bool, error) {
	if user.Role != "executor" {
		return false, nil
	}
	return s.repo.HasCoachProfile(ctx, user.ID)
}

func (s *Service) issueTokenPair(ctx context.Context, user User) (TokenPair, error) {
	isCoach, err := s.isCoachForToken(ctx, user)
	if err != nil {
		return TokenPair{}, err
	}
	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.Role, isCoach)
	if err != nil {
		return TokenPair{}, err
	}
	refreshID := uuid.NewString()
	refreshToken, err := s.jwt.GenerateRefreshToken(user.ID, user.Role, refreshID, isCoach)
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
