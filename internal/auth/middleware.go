package auth

import (
	"log/slog"
	"net/http"
	"strings"

	"opspilot/task-api/internal/platform/response"
)

// Authenticate returns middleware that enforces Bearer token verification and context claim population.
func Authenticate(tokenSvc TokenService, revStore RevocationStore, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				response.Error(w, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			tokenStr := strings.TrimSpace(parts[1])
			claims, err := tokenSvc.ValidateToken(tokenStr)
			if err != nil {
				if log != nil {
					log.Warn("token validation failed",
						"action", "auth.token_invalid",
						"error", err.Error(),
						"path", r.URL.Path,
					)
				}
				response.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			if revStore != nil && revStore.IsRevoked(claims.JTI) {
				if log != nil {
					log.Warn("revoked token access attempt",
						"action", "auth.token_revoked",
						"jti", claims.JTI,
						"path", r.URL.Path,
					)
				}
				response.Error(w, http.StatusUnauthorized, "token revoked")
				return
			}

			ctx := WithUser(r.Context(), *claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that restricts access to callers with designated roles.
func RequireRole(allowedRoles ...Role) func(http.Handler) http.Handler {
	roleMap := make(map[Role]bool, len(allowedRoles))
	for _, r := range allowedRoles {
		roleMap[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := UserFromContext(r.Context())
			if !ok {
				response.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			if !roleMap[claims.Role] {
				response.Error(w, http.StatusForbidden, "insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
