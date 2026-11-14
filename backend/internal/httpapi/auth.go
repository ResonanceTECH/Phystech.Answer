package httpapi

import (
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"phystech-answer/backend/internal/authx"
	"phystech-answer/backend/internal/model"
	"phystech-answer/backend/internal/store"
)

// Минимальная длина нового пароля (спека §4.2). Применяется только при
// установке пароля — на вход влияния не оказывает.
const minPasswordLen = 12

func validEmail(s string) bool {
	_, err := mail.ParseAddress(s)
	return err == nil && strings.Contains(s, "@")
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// setSessionCookie выставляет защищённую сессионную cookie.
func (s *Server) setSessionCookie(w http.ResponseWriter, token string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(ttl),
	})
}

// openSession создаёт серверную сессию: в БД — хеш токена, в cookie — сам токен.
func (s *Server) openSession(w http.ResponseWriter, r *http.Request, userID string, ttl time.Duration) bool {
	token, hash, err := authx.NewToken()
	if err != nil {
		writeErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create session")
		return false
	}
	if err := s.store.CreateSession(r.Context(), userID, hash, ttl); err != nil {
		writeErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create session")
		return false
	}
	s.setSessionCookie(w, token, ttl)
	return true
}

// ---------- login ----------

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.loginLimit.allow(clientIP(r)) {
		writeErrorCode(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many attempts, try later")
		return
	}
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	ident := strings.TrimSpace(req.Email)
	if ident == "" || req.Password == "" {
		writeErrorCode(w, http.StatusBadRequest, "VALIDATION_ERROR", "email and password are required")
		return
	}

	// Идентификатор — почта; логин поддерживается для ранних аккаунтов.
	var (
		user model.User
		hash string
		err  error
	)
	if strings.Contains(ident, "@") {
		user, hash, err = s.store.GetUserByEmail(r.Context(), ident)
	} else {
		user, hash, err = s.store.GetUserByLogin(r.Context(), ident)
	}
	if errors.Is(err, store.ErrNotFound) || (err == nil && !authx.VerifyPassword(hash, req.Password)) {
		// Единый ответ: по нему нельзя понять, существует ли аккаунт.
		writeErrorCode(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid credentials")
		return
	}
	if err != nil {
		writeErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "login failed")
		return
	}

	if active, err := s.store.IsUserActive(r.Context(), user.ID); err == nil && !active {
		writeError(w, http.StatusForbidden, "account is deactivated")
		return
	}
	ttl := s.sessionTTL
	if req.Remember {
		ttl = s.rememberTTL
	}
	if !s.openSession(w, r, user.ID, ttl) {
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// ---------- register ----------

type registerRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

// register создаёт аккаунт сотрудника и сразу открывает сессию.
// Системные права (role) клиентом не назначаются — всегда employee.
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if !s.registerLimit.allow(clientIP(r)) {
		writeErrorCode(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many attempts, try later")
		return
	}
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Email = normalizeEmail(req.Email)

	if req.FirstName == "" || req.LastName == "" {
		writeErrorCode(w, http.StatusBadRequest, "VALIDATION_ERROR", "first and last name are required")
		return
	}
	if !validEmail(req.Email) {
		writeErrorCode(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid email")
		return
	}
	if s.allowedEmailDomain != "" && !strings.HasSuffix(req.Email, "@"+s.allowedEmailDomain) {
		writeErrorCode(w, http.StatusBadRequest, "VALIDATION_ERROR", "email domain is not allowed")
		return
	}
	if len(req.Password) < minPasswordLen {
		writeErrorCode(w, http.StatusBadRequest, "VALIDATION_ERROR", "password must be at least 12 characters")
		return
	}

	hash, err := authx.HashPassword(req.Password)
	if err != nil {
		writeErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
		return
	}

	user, err := s.store.CreateUser(r.Context(), model.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Role:      "employee",
	}, hash)
	if errors.Is(err, store.ErrConflict) {
		writeErrorCode(w, http.StatusConflict, "EMAIL_ALREADY_EXISTS", "email already registered")
		return
	}
	if err != nil {
		writeErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create user")
		return
	}

	if !s.openSession(w, r, user.ID, s.sessionTTL) {
		return
	}
	writeJSON(w, http.StatusCreated, user)
}

// ---------- logout / me ----------

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_ = s.store.DeleteSession(r.Context(), authx.HashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
		Secure: s.secureCookies, SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, ok := s.currentUser(r)
	if !ok {
		writeErrorCode(w, http.StatusUnauthorized, "UNAUTHORIZED", "not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// ---------- forgot password ----------

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// forgotPassword создаёт одноразовый токен и отправляет письмо. Ответ
// одинаковый для существующих и неизвестных адресов — без утечки факта
// регистрации почты.
func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	if !s.forgotLimit.allow(clientIP(r)) {
		writeErrorCode(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests, try later")
		return
	}
	var req forgotPasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	email := normalizeEmail(req.Email)
	if !validEmail(email) {
		writeErrorCode(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid email")
		return
	}

	user, _, err := s.store.GetUserByEmail(r.Context(), email)
	if err == nil {
		token, hash, terr := authx.NewToken()
		if terr != nil {
			writeErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create token")
			return
		}
		if err := s.store.CreatePasswordResetToken(r.Context(), user.ID, hash, s.resetTTL); err != nil {
			writeErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create token")
			return
		}
		// Домен ссылки — из серверной конфигурации, не из заголовка Host.
		link := s.appPublicURL + "/reset-password?token=" + token
		text, html := resetEmailContent(link, s.resetTTL)
		if s.mailer.Enabled() {
			if err := s.mailer.Send(email, "Физтех.Ответ: восстановление пароля", text, html); err != nil {
				log.Printf("forgot-password: send to %s failed: %v", email, err)
			}
		} else {
			log.Printf("forgot-password: SMTP not configured, reset link for %s not delivered", email)
		}
	} else if !errors.Is(err, store.ErrNotFound) {
		writeErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "request failed")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "Если аккаунт существует, письмо отправлено",
	})
}

// ---------- reset password ----------

type resetPasswordRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	if !s.resetLimit.allow(clientIP(r)) {
		writeErrorCode(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests, try later")
		return
	}
	var req resetPasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Token) == "" {
		writeErrorCode(w, http.StatusBadRequest, "RESET_TOKEN_INVALID", "reset token is required")
		return
	}
	if len(req.NewPassword) < minPasswordLen {
		writeErrorCode(w, http.StatusBadRequest, "VALIDATION_ERROR", "password must be at least 12 characters")
		return
	}

	// Сначала атомарно гасим токен — параллельный запрос увидит used_at.
	user, err := s.store.ConsumePasswordResetToken(r.Context(), authx.HashToken(req.Token))
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErrorCode(w, http.StatusBadRequest, "RESET_TOKEN_INVALID", "reset link is invalid")
		return
	case errors.Is(err, store.ErrTokenExpired):
		writeErrorCode(w, http.StatusBadRequest, "RESET_TOKEN_EXPIRED", "reset link expired")
		return
	case errors.Is(err, store.ErrTokenUsed):
		writeErrorCode(w, http.StatusBadRequest, "RESET_TOKEN_USED", "reset link already used")
		return
	case err != nil:
		writeErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "reset failed")
		return
	}

	hash, err := authx.HashPassword(req.NewPassword)
	if err != nil {
		writeErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
		return
	}
	if err := s.store.UpdateUserPassword(r.Context(), user.ID, hash); err != nil {
		writeErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "reset failed")
		return
	}
	// Все существующие сессии пользователя отзываются.
	_ = s.store.RevokeUserSessions(r.Context(), user.ID)

	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

// ---------- profile / departments ----------

// patchMe обновляет разрешённые поля профиля текущего пользователя.
// Роль и системные разрешения через этот endpoint не меняются.
func (s *Server) patchMe(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var patch model.ProfilePatch
	if !decodeJSON(w, r, &patch) {
		return
	}
	if patch.FirstName != nil && strings.TrimSpace(*patch.FirstName) == "" {
		writeErrorCode(w, http.StatusBadRequest, "VALIDATION_ERROR", "first name cannot be empty")
		return
	}
	if patch.LastName != nil && strings.TrimSpace(*patch.LastName) == "" {
		writeErrorCode(w, http.StatusBadRequest, "VALIDATION_ERROR", "last name cannot be empty")
		return
	}

	updated, err := s.store.UpdateUserProfile(r.Context(), u.ID, patch)
	if err != nil {
		writeErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update profile")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) listDepartments(w http.ResponseWriter, r *http.Request) {
	depts, err := s.store.ListDepartments(r.Context())
	if err != nil {
		writeErrorCode(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list departments")
		return
	}
	writeJSON(w, http.StatusOK, depts)
}
