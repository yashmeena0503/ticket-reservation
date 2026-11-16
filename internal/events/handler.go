package events

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"reservation/internal/auth"
	"reservation/internal/db"
	"reservation/internal/platform/database"
	"reservation/internal/platform/httpx"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type handler struct {
	service *auth.Service
	q *db.Queries
	pool *pgxpool.Pool
}

// func Register(mux *http.ServeMux, service *Service) {
func Register(mux *http.ServeMux, pool *pgxpool.Pool, service *auth.Service) {
	h := &handler{ q: db.New(pool), pool: pool, service: service }
	mux.HandleFunc("GET /event", h.list)
	mux.HandleFunc("GET /event/{id}", h.get)
	mux.Handle("POST /event", httpx.RequireAuth(service.Verify, http.HandlerFunc(h.new)))
	mux.Handle("POST /event/{id}/section", httpx.RequireAuth(service.Verify, http.HandlerFunc(h.newSection)))
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

func (h *handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	event, err := h.q.GetEvent(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "event not found")
		return
	}
	if err != nil {
		slog.Error("get event", "request_id", httpx.RequestIdFrom(r.Context()), "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	sections, err := h.q.ListSectionsWithAvailability(r.Context(), id)
	if err != nil {
		slog.Error("list sections with availability", "request_id", httpx.RequestIdFrom(r.Context()), "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httpx.WriteJson(w, http.StatusOK, struct {
		db.Event
		Sections []db.ListSectionsWithAvailabilityRow `json:"sections"`
	}{event, sections})
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

func (h *handler) newSection(w http.ResponseWriter, r *http.Request) {
	userId, _ := httpx.UserFrom(r.Context())
	eventId, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req struct {
		Name string `json:"name"`
		Price int32 `json:"price"`
		Rows int `json:"rows"`
		SeatsPerRow int `json:"seats_per_row"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	// rows capped at 26 because row labels are single letters A-Z
	if strings.TrimSpace(req.Name) == "" ||
	req.Price < 0||
	req.Rows < 1 ||
	req.Rows > 26 ||
	req.SeatsPerRow < 1 ||
	req.SeatsPerRow > 100 {
		httpx.WriteError(w, http.StatusBadRequest, "name, price >= 0, rows (1-26), seats_per_row (1-100) required")
		return
	}

	// ownership check
	event, err := h.q.GetEvent(r.Context(), eventId)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "event not found")
		return
	}
	if err != nil {
		slog.Error("new section: get event", "request_id", httpx.RequestIdFrom(r.Context()), "err", err)
		httpx.WriteError(w, http.StatusNotFound, "event not found")
		return
	}
	if event.OrganizerID != userId {
		httpx.WriteError(w, http.StatusForbidden, "you are not the event organizer")
		return
	}

	// section + all its seats in one transaction
	var section db.Section
	err = database.WithTx(r.Context(), h.pool, func(tx pgx.Tx) error {
		q := h.q.WithTx(tx)
		var err error
		section, err = q.NewSection(r.Context(), db.NewSectionParams{
			EventID: eventId,
			Name: req.Name,
			Price: req.Price,
		})
		if err != nil {
			return err
		}
		seats := make([]db.NewSeatParams, 0, req.Rows*req.SeatsPerRow)
		for row := 0; row < req.Rows; row++ {
			for n := 1; n <= req.SeatsPerRow; n++ {
				seats = append(seats, db.NewSeatParams{
					SectionID: section.ID,
					RowLabel: string(rune('A' + row)), // A, B, C...
					SeatNumber: int32(n),
				})
			}
		}
		_, err = q.NewSeat(r.Context(), seats)
		return err
	})

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // UNIQUE (event_id, name)
		httpx.WriteError(w, http.StatusConflict, "section name already exists for this event")
		return
	}
	if err != nil {
		slog.Error("new section", "request_id", httpx.RequestIdFrom(r.Context()), "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
		return
	}

	httpx.WriteJson(w, http.StatusCreated, section)
}
