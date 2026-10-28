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

	// Authenticated logout
	authMw := Authenticate(tokenSvc, revStore, log)
	mux.Handle("POST /auth/logout", authMw(http.HandlerFunc(handler.Logout)))

	return mux
}
