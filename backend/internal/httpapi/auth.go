package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"phystech-answer/backend/internal/model"
	"phystech-answer/backend/internal/store"
)

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Login = strings.TrimSpace(req.Login)
	if req.Login == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "login and password are required")
		return
	}

	user, hash, err := s.store.GetUserByLogin(r.Context(), req.Login)
	if errors.Is(err, store.ErrNotFound) || (err == nil && bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password)) != nil) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "login failed")
		return
	}

	sessionID, err := s.store.CreateSession(r.Context(), user.ID, s.sessionTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(s.sessionTTL),
	})
	writeJSON(w, http.StatusOK, user)
}

type registerRequest struct {
	Name     string `json:"name"`
	Login    string `json:"login"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// register создаёт пользователя с ролью employee и сразу открывает сессию.
// Административные роли выдаются только через базу/админку — самоназначить
// их через регистрацию нельзя.
func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Login = strings.TrimSpace(req.Login)
	req.Email = strings.TrimSpace(req.Email)
	if req.Name == "" || req.Login == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "name, login and password are required")
		return
	}
	if len(req.Password) < 6 {
		writeError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := s.store.CreateUser(r.Context(), model.User{
		Name:  req.Name,
		Email: req.Email,
		Login: req.Login,
		Role:  "employee",
	}, string(hash))
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "login already taken")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	sessionID, err := s.store.CreateSession(r.Context(), user.ID, s.sessionTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create session")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(s.sessionTTL),
	})
	writeJSON(w, http.StatusCreated, user)
}

type forgotPasswordRequest struct {
	Login string `json:"login"`
}

// forgotPassword — заглушка: выдача reset-токена и отправка почты ещё не
// подключены. Ответ одинаковый независимо от существования аккаунта, чтобы
// не раскрывать, какие логины заняты.
func (s *Server) forgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Login) == "" {
		writeError(w, http.StatusBadRequest, "login is required")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"ok": true})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_ = s.store.DeleteSession(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, ok := s.currentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	writeJSON(w, http.StatusOK, u)
}
