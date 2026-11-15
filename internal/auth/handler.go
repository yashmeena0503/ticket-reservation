package auth

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"reservation/internal/platform/httpx"
	"strings"
)

type handler struct {
	service *Service
}

func Register(mux *http.ServeMux, service *Service) {
	h := &handler{service: service}
	mux.HandleFunc("POST /auth/register", h.register)
	mux.HandleFunc("POST /auth/login", h.login)
	mux.Handle("GET /auth/me", httpx.RequireAuth(service.Verify, http.HandlerFunc(h.me)))
}

type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func decode(w http.ResponseWriter, r *http.Request) (credentials, bool) {
	var creds credentials
	r.Body = http.MaxBytesReader(w, r.Body, 1<<10)
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid body")
		return creds, false
	}
	creds.Email = strings.ToLower(strings.TrimSpace(creds.Email))
	return creds, true
}

func (h *handler) register(w http.ResponseWriter, r *http.Request) {
	creds, ok := decode(w, r)
	if !ok {
		return
	}
	// hard limit on password length
	addr, err := mail.ParseAddress(creds.Email)
	if err != nil || addr.Address != creds.Email || len(creds.Password) < 8 || len(creds.Password) > 128 {
		httpx.WriteError(w, http.StatusBadRequest, "valid email and password (8-128 characters) required")
		return
	}
	user, err := h.service.Register(r.Context(), creds.Email, creds.Password)
	if errors.Is(err, ErrEmailTaken) {
		httpx.WriteError(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		slog.Error("register", "request_id", httpx.RequestIdFrom(r.Context()), "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpx.WriteJson(w, http.StatusCreated, user)
}

func (h *handler) login(w http.ResponseWriter, r *http.Request) {
	creds, ok := decode(w, r)
	if !ok {
		return
	}
	token, err := h.service.Login(r.Context(), creds.Email, creds.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		slog.Error("login", "request_id", httpx.RequestIdFrom(r.Context()), "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpx.WriteJson(w, http.StatusOK, map[string]string{"access_token": token})
}

func (h *handler) me(w http.ResponseWriter, r *http.Request) {
	id, ok := httpx.UserFrom(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}
	httpx.WriteJson(w, http.StatusOK, map[string]string{"id": id.String()})
}
