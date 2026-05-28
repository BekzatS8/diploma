package middleware

import (
	"net/http"
	"strings"

	"buhpro/internal/common/auth"
	"buhpro/internal/common/response"

	"github.com/gin-gonic/gin"
)

const bearerPrefix = "Bearer "

type UserContext struct {
	UserID string
	Roles  []string
}

func (u UserContext) PrimaryRole() string {
	if len(u.Roles) == 0 {
		return ""
	}
	return u.Roles[0]
}

func CurrentUser(c *gin.Context) (UserContext, bool) {
	v, ok := c.Get("auth_user")
	if !ok {
		return UserContext{}, false
	}
	user, ok := v.(UserContext)
	return user, ok
}

func OptionalAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c.GetHeader("Authorization"))
		if token == "" {
			c.Next()
			return
		}

		claims, err := jwtManager.ParseAccessToken(token)
		if err != nil {
			c.Next()
			return
		}

		c.Set("auth_user", userContextFromClaims(claims))
		c.Next()
	}
}

func RequireAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c.GetHeader("Authorization"))
		if token == "" {
			response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Missing access token")
			return
		}

		claims, err := jwtManager.ParseAccessToken(token)
		if err != nil {
			response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Invalid or expired token")
			return
		}

		c.Set("auth_user", userContextFromClaims(claims))
		c.Next()
	}
}

func userContextFromClaims(claims *auth.Claims) UserContext {
	roles := []string{claims.Role}
	if claims.IsCoach && claims.Role != "coach" {
		roles = append(roles, "coach")
	}
	return UserContext{UserID: claims.UserID, Roles: roles}
}

func RequireRoles(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, role := range allowed {
		allowedSet[role] = struct{}{}
	}

	return func(c *gin.Context) {
		ctxUser, ok := CurrentUser(c)
		if !ok {
			response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}

		for _, role := range ctxUser.Roles {
			if _, exists := allowedSet[role]; exists {
				c.Next()
				return
			}
		}

		response.JSONError(c, http.StatusForbidden, "forbidden", "Insufficient role")
	}
}

func extractBearer(header string) string {
	if !strings.HasPrefix(header, bearerPrefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(header, bearerPrefix))
}
