package app

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/subscriptions", h.Create)
	r.Get("/subscriptions", h.List)
	r.Get("/subscriptions/total", h.TotalCost) // must be before /{id}
	r.Get("/subscriptions/{id}", h.GetByID)
	r.Put("/subscriptions/{id}", h.Update)
	r.Delete("/subscriptions/{id}", h.Delete)

	return r
}

// @Summary      Create subscription
// @Description  Add a new subscription record
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        input  body      CreateSubscriptionInput  true  "Subscription data"
// @Success      201    {object}  Subscription
// @Failure      400    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Router       /subscriptions [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var in CreateSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sub, err := h.svc.Create(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, sub)
}

// @Summary      Get subscription
// @Description  Get a single subscription by ID
// @Tags         subscriptions
// @Produce      json
// @Param        id   path      string  true  "Subscription UUID"
// @Success      200  {object}  Subscription
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /subscriptions/{id} [get]
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	sub, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

// @Summary      List subscriptions
// @Description  List all subscriptions, optionally filtered by user_id or service_name
// @Tags         subscriptions
// @Produce      json
// @Param        user_id       query     string  false  "Filter by user UUID"
// @Param        service_name  query     string  false  "Filter by service name"
// @Success      200           {array}   Subscription
// @Failure      500           {object}  ErrorResponse
// @Router       /subscriptions [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	serviceName := r.URL.Query().Get("service_name")

	subs, err := h.svc.List(r.Context(), userID, serviceName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if subs == nil {
		subs = []Subscription{}
	}
	writeJSON(w, http.StatusOK, subs)
}

// @Summary      Update subscription
// @Description  Update service_name, price, or end_date of an existing subscription
// @Tags         subscriptions
// @Accept       json
// @Produce      json
// @Param        id     path      string                  true  "Subscription UUID"
// @Param        input  body      UpdateSubscriptionInput  true  "Fields to update"
// @Success      200    {object}  Subscription
// @Failure      400    {object}  ErrorResponse
// @Failure      404    {object}  ErrorResponse
// @Failure      500    {object}  ErrorResponse
// @Router       /subscriptions/{id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var in UpdateSubscriptionInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	sub, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

// @Summary      Delete subscription
// @Description  Remove a subscription record by ID
// @Tags         subscriptions
// @Param        id   path  string  true  "Subscription UUID"
// @Success      204  "No Content"
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /subscriptions/{id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary      Total subscription cost
// @Description  Calculate the total cost of subscriptions over a time period
// @Tags         subscriptions
// @Produce      json
// @Param        from          query     string  true   "Period start (MM-YYYY)"
// @Param        to            query     string  true   "Period end (MM-YYYY)"
// @Param        user_id       query     string  false  "Filter by user UUID"
// @Param        service_name  query     string  false  "Filter by service name"
// @Success      200           {object}  TotalCostResponse
// @Failure      400           {object}  ErrorResponse
// @Failure      500           {object}  ErrorResponse
// @Router       /subscriptions/total [get]
func (h *Handler) TotalCost(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := TotalCostFilter{
		From:        q.Get("from"),
		To:          q.Get("to"),
		UserID:      q.Get("user_id"),
		ServiceName: q.Get("service_name"),
	}

	total, err := h.svc.TotalCost(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, TotalCostResponse{Total: total})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, ErrorResponse{Error: msg})
}
