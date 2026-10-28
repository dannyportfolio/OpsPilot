package auth

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	"opspilot/task-api/internal/platform/response"
)

// Handler processes authentication transport requests.
type Handler struct {
	service Service
	logger  *slog.Logger
}

// NewHandler constructs an auth HTTP handler.
func NewHandler(service Service, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// Login handles POST /auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 1 MB maximum payload limit
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var req LoginRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := dec.Decode(&struct{}{}); err != io.EOF {
		response.Error(w, http.StatusBadRequest, "request body must contain only a single JSON object")
		return
	}

	if strings.TrimSpace(req.Email) == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	clientIP := extractClientIP(r)

	resp, err := h.service.Login(r.Context(), req, clientIP)
	if err != nil {
		switch {
		case errors.Is(err, ErrRateLimitExceeded):
			response.Error(w, http.StatusTooManyRequests, "too many login attempts, please try again later")
		case errors.Is(err, ErrUserSuspended):
			response.Error(w, http.StatusForbidden, "account suspended")
		case errors.Is(err, ErrInvalidCredentials):
			response.Error(w, http.StatusUnauthorized, "invalid credentials")
		default:
			if h.logger != nil {
				h.logger.Error("internal error during login", "error", err)
			}
			response.Error(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// Logout handles POST /auth/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := UserFromContext(r.Context())
	if !ok {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.service.Logout(r.Context(), claims); err != nil {
		if h.logger != nil {
			h.logger.Error("internal error during logout", "error", err)
		}
		response.Error(w, http.StatusInternalServerError, "failed to revoke token")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}

// extractClientIP extracts the remote client IP from headers or RemoteAddr.
func extractClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ips := strings.Split(fwd, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if parsed := net.ParseIP(ip); parsed != nil {
				return ip
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}
