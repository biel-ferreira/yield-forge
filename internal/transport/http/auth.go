package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/biel-ferreira/yield-forge/internal/auth"
)

// AuthService is the slice of the auth service the transport layer needs. Defining
// it here (consumer-side) keeps handlers and middleware testable with a small fake;
// *auth.Service satisfies it (SPEC-003 FR-301..305).
type AuthService interface {
	Register(ctx context.Context, email, password string) (auth.User, error)
	Login(ctx context.Context, email, password string) (auth.User, string, error)
	Logout(ctx context.Context, rawToken string) error
	Authenticate(ctx context.Context, rawToken string) (auth.User, error)
	GetUserByID(ctx context.Context, id string) (auth.User, error)
}

// authHandler serves the /auth endpoints and owns session-cookie shaping.
type authHandler struct {
	service      AuthService
	logger       *slog.Logger
	cookieName   string
	cookieSecure bool
	sessionTTL   time.Duration
}

// credentialsRequest is the body for register and login.
type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// userResponse is the public view of a user — never includes the password hash.
type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// register creates a new account (SPEC-003 FR-301).
func (h authHandler) register(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.service.Register(r.Context(), req.Email, req.Password)
	switch {
	case errors.Is(err, auth.ErrEmailTaken):
		writeError(w, http.StatusConflict, "email already registered")
		return
	case errors.Is(err, auth.ErrInvalidEmail):
		writeError(w, http.StatusBadRequest, "invalid email address")
		return
	case errors.Is(err, auth.ErrWeakPassword):
		writeError(w, http.StatusBadRequest,
			fmt.Sprintf("password must be at least %d characters", auth.MinPasswordLength))
		return
	case errors.Is(err, auth.ErrPasswordTooLong):
		writeError(w, http.StatusBadRequest,
			fmt.Sprintf("password must be at most %d bytes", auth.MaxPasswordLength))
		return
	case err != nil:
		h.logger.ErrorContext(r.Context(), "register failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, userResponse{ID: user.ID, Email: user.Email})
}

// login verifies credentials and issues a session cookie (SPEC-003 FR-302).
func (h authHandler) login(w http.ResponseWriter, r *http.Request) {
	var req credentialsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, token, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			h.logger.WarnContext(r.Context(), "login failed", slog.String("email", maskEmail(req.Email)))
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		h.logger.ErrorContext(r.Context(), "login error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	h.setSessionCookie(w, token)
	h.logger.InfoContext(r.Context(), "login", slog.String("user_id", user.ID))
	writeJSON(w, http.StatusOK, userResponse{ID: user.ID, Email: user.Email})
}

// logout revokes the current session and clears the cookie (SPEC-003 FR-303). It is
// a protected route, so it only runs for an authenticated request.
func (h authHandler) logout(w http.ResponseWriter, r *http.Request) {
	token := sessionTokenFromRequest(r, h.cookieName)
	if err := h.service.Logout(r.Context(), token); err != nil {
		h.logger.ErrorContext(r.Context(), "logout error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// me returns the authenticated caller's identity (SPEC-003 FR-304). Identity comes
// from the context the middleware set, never from request input (BR-304).
func (h authHandler) me(w http.ResponseWriter, r *http.Request) {
	id, ok := auth.UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	user, err := h.service.GetUserByID(r.Context(), id)
	if err != nil {
		// A valid session whose user no longer exists is treated as unauthenticated.
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	writeJSON(w, http.StatusOK, userResponse{ID: user.ID, Email: user.Email})
}

// setSessionCookie writes the session cookie: HttpOnly + SameSite always, Secure
// outside dev (SPEC-003 D3 / §10).
func (h authHandler) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(h.sessionTTL.Seconds()),
	})
}

// clearSessionCookie expires the session cookie on the client (logout).
func (h authHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     h.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// sessionTokenFromRequest reads the raw session token from the session cookie, or ""
// if absent.
func sessionTokenFromRequest(r *http.Request, cookieName string) string {
	c, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return c.Value
}

// maskEmail partially redacts an email for logs (e.g. alice@x.com -> a***@x.com),
// so failed-login logs never store the full address (SPEC-003 §11).
func maskEmail(email string) string {
	at := strings.IndexByte(email, '@')
	if at <= 1 {
		return "***"
	}
	return email[:1] + "***" + email[at:]
}
