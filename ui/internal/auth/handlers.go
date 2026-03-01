package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/axonops/schema-registry-ui/internal/htpasswd"
)

// Handlers provides HTTP handlers for authentication endpoints.
type Handlers struct {
	htpasswd     *htpasswd.File
	tokenManager *TokenManager
	cookieName   string
	cookieSecure bool
}

// NewHandlers creates auth handlers.
func NewHandlers(hp *htpasswd.File, tm *TokenManager, cookieName string, cookieSecure bool) *Handlers {
	return &Handlers{
		htpasswd:     hp,
		tokenManager: tm,
		cookieName:   cookieName,
		cookieSecure: cookieSecure,
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Username string `json:"username"`
}

type sessionResponse struct {
	Username string `json:"username"`
}

type configResponse struct {
	AuthEnabled bool `json:"auth_enabled"`
}

// Login handles POST /api/auth/login.
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, `{"error":"username and password required"}`, http.StatusBadRequest)
		return
	}

	ok, err := h.htpasswd.Verify(req.Username, req.Password)
	if err != nil {
		slog.Error("htpasswd verify failed", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	token, expiresAt, err := h.tokenManager.Create(req.Username)
	if err != nil {
		slog.Error("token creation failed", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	slog.Info("user logged in", "username", req.Username)
	writeJSON(w, http.StatusOK, loginResponse{Username: req.Username})
}

// Logout handles POST /api/auth/logout.
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	username := UsernameFromContext(r.Context())

	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	slog.Info("user logged out", "username", username)
	w.WriteHeader(http.StatusNoContent)
}

// Session handles GET /api/auth/session.
// Returns the current user's session info and refreshes the token.
func (h *Handlers) Session(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	username := UsernameFromContext(r.Context())
	if username == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Refresh the token on each session check
	token, expiresAt, err := h.tokenManager.Create(username)
	if err != nil {
		slog.Error("token refresh failed", "error", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
	})

	writeJSON(w, http.StatusOK, sessionResponse{Username: username})
}

// Config handles GET /api/auth/config.
// Returns authentication configuration (always enabled for standalone UI).
func (h *Handlers) Config(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, configResponse{AuthEnabled: true})
}

// Health handles GET /health for the UI server itself.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "UP",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
