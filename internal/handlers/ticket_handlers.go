package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"ticket-system/internal/models"
	"ticket-system/internal/store"
)

type TicketHandler struct {
	store *store.Store
}

func NewTicketHandler(s *store.Store) *TicketHandler {
	return &TicketHandler{store: s}
}

type createTicketRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (h *TicketHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	ticket := h.store.CreateTicket(userID, req.Title, req.Description)
	writeJSON(w, http.StatusCreated, ticket)
}

func (h *TicketHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	tickets := h.store.ListTicketsByUser(userID)
	writeJSON(w, http.StatusOK, tickets)
}

func (h *TicketHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := r.PathValue("id")
	ticket, err := h.store.GetTicket(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}
	if ticket.UserID != userID {
		writeError(w, http.StatusForbidden, "you do not have access to this ticket")
		return
	}

	writeJSON(w, http.StatusOK, ticket)
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

// allowedTransitions encodes the required linear status flow:
// open -> in_progress -> closed, with closed being terminal.
var allowedTransitions = map[models.TicketStatus]models.TicketStatus{
	models.StatusOpen:       models.StatusInProgress,
	models.StatusInProgress: models.StatusClosed,
}

func (h *TicketHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := r.PathValue("id")
	ticket, err := h.store.GetTicket(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}
	if ticket.UserID != userID {
		writeError(w, http.StatusForbidden, "you do not have access to this ticket")
		return
	}

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	newStatus := models.TicketStatus(strings.ToLower(strings.TrimSpace(req.Status)))
	if !newStatus.IsValid() {
		writeError(w, http.StatusBadRequest, "status must be one of: open, in_progress, closed")
		return
	}

	if ticket.Status == models.StatusClosed {
		writeError(w, http.StatusBadRequest, "a closed ticket cannot be reopened or modified")
		return
	}

	expectedNext, hasTransition := allowedTransitions[ticket.Status]
	if !hasTransition || expectedNext != newStatus {
		writeError(w, http.StatusBadRequest,
			"invalid status transition: must follow open -> in_progress -> closed")
		return
	}

	updated, err := h.store.UpdateTicketStatus(id, newStatus)
	if err != nil {
		writeError(w, http.StatusNotFound, "ticket not found")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}
