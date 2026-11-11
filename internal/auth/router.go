package auth

import (
	"log/slog"
	"net/http"
)

// NewRouter registers auth routes returning a decoupled http.Handler.
func NewRouter(svc Service, tokenSvc TokenService, revStore RevocationStore, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	handler := NewHandler(svc, log)

	// Public login
	mux.HandleFunc("POST /auth/login", handler.Login)

	// Authenticated middleware chain
	authMw := Authenticate(tokenSvc, revStore, log)
	adminMw := RequireRole(RoleAdmin)

	// Authenticated endpoints
	mux.Handle("POST /auth/logout", authMw(http.HandlerFunc(handler.Logout)))
	mux.Handle("GET /auth/me", authMw(http.HandlerFunc(handler.Me)))

	// Admin user provisioning
	mux.Handle("POST /auth/users", authMw(adminMw(http.HandlerFunc(handler.CreateUser))))

	return mux
}
