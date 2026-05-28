package auth

import (
	"errors"
	"net/http"

	"buhpro/internal/common/response"
	"buhpro/internal/http/middleware"
	profilemod "buhpro/internal/modules/profile"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service        *Service
	profileService *profilemod.Service
}

func NewHandler(service *Service, profileService *profilemod.Service) *Handler {
	return &Handler{service: service, profileService: profileService}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}

	user, pair, err := h.service.Register(c.Request.Context(), RegisterInput{
		Email:           req.Email,
		Password:        req.Password,
		Role:            req.Role,
		ProfileName:     req.ProfileName,
		Phone:           req.Phone,
		ClientType:      req.ClientType,
		TaxNumber:       req.TaxNumber,
		ContactName:     req.ContactName,
		ContactPosition: req.ContactPosition,
		Address:         req.Address,
		About:           req.About,
		Website:         req.Website,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		MiddleName:      req.MiddleName,
		IIN:             req.IIN,
		City:            req.City,
		ExperienceLevel: req.ExperienceLevel,
		Specializations: req.Specializations,
		Education:       req.Education,
		WorkFormat:      req.WorkFormat,
		HourlyRate:      req.HourlyRate,
	})
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	response.JSON(c, http.StatusCreated, AuthResponse{
		UserID:             user.ID,
		Email:              user.Email,
		Role:               user.Role,
		VerificationStatus: user.VerificationStatus,
		AccessToken:        pair.AccessToken,
		RefreshToken:       pair.RefreshToken,
	})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}

	user, pair, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	response.JSON(c, http.StatusOK, AuthResponse{
		UserID:             user.ID,
		Email:              user.Email,
		Role:               user.Role,
		VerificationStatus: user.VerificationStatus,
		AccessToken:        pair.AccessToken,
		RefreshToken:       pair.RefreshToken,
	})
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}

	pair, err := h.service.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	response.JSON(c, http.StatusOK, pair)
}

func (h *Handler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}

	user, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	if err := h.service.Logout(c.Request.Context(), user.UserID, req.RefreshToken); err != nil {
		h.handleAuthError(c, err)
		return
	}

	response.JSON(c, http.StatusOK, gin.H{"status": "logged_out"})
}

func (h *Handler) Me(c *gin.Context) {
	userCtx, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	user, err := h.service.Me(c.Request.Context(), userCtx.UserID)
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	profile, err := h.profileService.GetCurrentProfile(c.Request.Context(), user.ID, user.Role)
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to load profile")
		return
	}

	isCoach, err := h.service.IsCoachCapability(c.Request.Context(), user.ID, user.Role)
	if err != nil {
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to load user capabilities")
		return
	}

	response.JSON(c, http.StatusOK, gin.H{
		"id":                  user.ID,
		"email":               user.Email,
		"role":                user.Role,
		"is_coach":            isCoach,
		"verification_status": user.VerificationStatus,
		"profile":             profile,
	})
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}

	resp, err := h.service.RequestPasswordReset(c.Request.Context(), req.Email)
	if err != nil {
		h.handleAuthError(c, err)
		return
	}

	response.JSON(c, http.StatusOK, resp)
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}

	if err := h.service.ResetPasswordWithToken(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		h.handleAuthError(c, err)
		return
	}

	response.JSON(c, http.StatusOK, gin.H{
		"status":  "password_reset",
		"message": "Пароль успешно обновлён",
	})
}

func (h *Handler) ChangePassword(c *gin.Context) {
	userCtx, ok := middleware.CurrentUser(c)
	if !ok {
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.JSONError(c, http.StatusBadRequest, "bad_request", "Invalid request payload")
		return
	}

	if err := h.service.ChangePassword(
		c.Request.Context(),
		userCtx.UserID,
		req.CurrentPassword,
		req.NewPassword,
	); err != nil {
		h.handleAuthError(c, err)
		return
	}

	response.JSON(c, http.StatusOK, gin.H{"status": "password_updated"})
}

func (h *Handler) handleAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrEmailAlreadyExists):
		response.JSONError(c, http.StatusConflict, "email_exists", "Email already in use")
	case errors.Is(err, ErrInvalidRole):
		response.JSONError(c, http.StatusBadRequest, "invalid_role", "Unsupported role")
	case errors.Is(err, ErrExecutorLeadRequired):
		response.JSONError(c, http.StatusBadRequest, "executor_lead_required", "Executor registration requires document lead verification")
	case errors.Is(err, ErrInvalidCredentials):
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Invalid email or password")
	case errors.Is(err, ErrWrongCurrentPassword):
		response.JSONError(c, http.StatusBadRequest, "wrong_password", "Current password is incorrect")
	case errors.Is(err, ErrSamePassword):
		response.JSONError(c, http.StatusBadRequest, "same_password", "New password must differ from the current password")
	case errors.Is(err, ErrInvalidResetToken):
		response.JSONError(c, http.StatusBadRequest, "invalid_reset_token", "Invalid or expired password reset link")
	case errors.Is(err, ErrUnauthorized):
		response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
	default:
		if err != nil && err.Error() != "" {
			if err.Error() == "password must be at least 8 characters" {
				response.JSONError(c, http.StatusBadRequest, "invalid_password", err.Error())
				return
			}
		}
		response.JSONError(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
