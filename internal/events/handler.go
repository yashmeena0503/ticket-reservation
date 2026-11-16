package events

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"reservation/internal/auth"
	"reservation/internal/db"
	"reservation/internal/platform/httpx"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type handler struct {
	service *auth.Service
	q *db.Queries
}

// func Register(mux *http.ServeMux, service *Service) {
func Register(mux *http.ServeMux, q *db.Queries, service *auth.Service) {
	h := &handler{ q: q, service: service }
	mux.HandleFunc("GET /events", h.list)
	mux.Handle("POST /events", httpx.RequireAuth(service.Verify, http.HandlerFunc(h.new)))
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := 20 // default value
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = n
	}

	afterTime := time.Time{}
	if value := q.Get("after_starts_at"); value != "" {
		t, err := time.Parse(time.RFC3339Nano, value)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid after_starts_at")
			return
		}
		afterTime = t
	}

	afterId := uuid.Nil
	if value := q.Get("after_id"); value != "" {
		id, err := uuid.Parse(value)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid after_id")
			return
		}
		afterId = id
	}

	events, err := h.q.ListEventsAfter(r.Context(), db.ListEventsAfterParams{
		AfterStartsAt: afterTime,
		AfterID: afterId,
		PageSize: int32(limit),
	})
	if err != nil {
		slog.Error("list events", "request_id", httpx.RequestIdFrom(r.Context()), "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httpx.WriteJson(w, http.StatusOK, events)
}

func (h *handler) new(w http.ResponseWriter, r *http.Request) {
	userId, _ := httpx.UserFrom(r.Context())

	var req struct {
		Name string `json:"name"`
		Description string `json:"description"` // can be null
		Venue string `json:"venue"`
		StartsAt time.Time `json:"starts_at"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	// skipped description because it can be null
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Venue) == "" || req.StartsAt.IsZero() {
		httpx.WriteError(w, http.StatusBadRequest, "name, venue and starts_at are required")
		return
	}

	event, err := h.q.NewEvent(r.Context(), db.NewEventParams{
		OrganizerID: userId,
		Name: req.Name,
		Description: req.Description,
		Venue: req.Venue,
		StartsAt: req.StartsAt,
	})
	if err != nil {
		slog.Error("new event", "request_id", httpx.RequestIdFrom(r.Context()), "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httpx.WriteJson(w, http.StatusCreated, event)
}
