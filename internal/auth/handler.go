package auth

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"

	"opspilot/task-api/internal/platform/httputil"
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

	var req LoginRequest
	if err := httputil.DecodeJSON(w, r, &req, httputil.DefaultMaxBodyBytes); err != nil {
		if strings.Contains(err.Error(), "multiple JSON values") {
			response.Error(w, http.StatusBadRequest, "request body must contain only a single JSON object")
			return
		}
		response.Error(w, http.StatusBadRequest, "invalid request body")
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
		case errors.Is(err, ErrAmbiguousUser):
			response.Error(w, http.StatusBadRequest, "multiple accounts found with this email; please provide tenant_id")
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

// CreateUserRequest contains fields required to register a tenant user.
type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
}

// CreateUser handles POST /auth/users (admin only within tenant).
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := UserFromContext(r.Context())
	if !ok || claims.TenantID == "" {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req CreateUserRequest
	if err := httputil.DecodeJSON(w, r, &req, httputil.DefaultMaxBodyBytes); err != nil {
		if strings.Contains(err.Error(), "multiple JSON values") {
			response.Error(w, http.StatusBadRequest, "request body must contain only a single JSON object")
			return
		}
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.Email) == "" || req.Password == "" {
		response.Error(w, http.StatusBadRequest, "email and password are required")
		return
	}

	if req.Role == "" {
		req.Role = RoleMember
	}

	created, err := h.service.CreateUser(r.Context(), claims.TenantID, req.Email, req.Password, req.Role)
	if err != nil {
		if errors.Is(err, ErrUserAlreadyExists) {
			response.Error(w, http.StatusConflict, "user with this email already exists")
			return
		}
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, created)
}

// Me handles GET /auth/me (returns profile of current authenticated user).
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	claims, ok := UserFromContext(r.Context())
	if !ok || claims.Subject == "" {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.service.GetProfile(r.Context(), claims.Subject)
	if err != nil {
		response.Error(w, http.StatusNotFound, "user profile not found")
		return
	}

	response.JSON(w, http.StatusOK, user)
}

// isTrustedProxy reports whether an IP string is a trusted peer (loopback or configured trusted proxy).
func isTrustedProxy(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	if proxies := os.Getenv("TRUSTED_PROXIES"); proxies != "" {
		for _, p := range strings.Split(proxies, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if p == ipStr {
				return true
			}
			if _, cidr, err := net.ParseCIDR(p); err == nil && cidr.Contains(ip) {
				return true
			}
		}
	}
	return false
}

// extractClientIP safely resolves the client IP address.
// X-Forwarded-For is strictly inspected ONLY when the direct network peer (RemoteAddr)
// is verified to be a trusted proxy (e.g. loopback or configured TRUSTED_PROXIES).
// Otherwise, the direct network peer address is returned to prevent IP spoofing attacks.
func extractClientIP(r *http.Request) string {
	remoteHost, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remoteHost = r.RemoteAddr
	}

	if isTrustedProxy(remoteHost) {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ips := strings.Split(fwd, ",")
			if len(ips) > 0 {
				clientIP := strings.TrimSpace(ips[0])
				if parsed := net.ParseIP(clientIP); parsed != nil {
					return clientIP
				}
			}
		}
	}

	return remoteHost
}
