package store

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"ticket-system/internal/models"
)

var (
	ErrUserExists     = errors.New("user already exists")
	ErrUserNotFound   = errors.New("user not found")
	ErrTicketNotFound = errors.New("ticket not found")
)

// Store is a simple thread-safe in-memory store for users and tickets.
// The assignment scope explicitly allows in-memory storage; this keeps the
// service self-contained with zero external dependencies (no DB driver
// needed), which also simplifies the Docker build and deployment.
type Store struct {
	mu sync.RWMutex

	usersByID    map[string]*models.User
	usersByEmail map[string]*models.User
	tickets      map[string]*models.Ticket

	nextUserID   int
	nextTicketID int
}

func New() *Store {
	return &Store{
		usersByID:    make(map[string]*models.User),
		usersByEmail: make(map[string]*models.User),
		tickets:      make(map[string]*models.Ticket),
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// CreateUser stores a new user with the given (already-hashed) password.
func (s *Store) CreateUser(email, passwordHash string) (*models.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	email = normalizeEmail(email)
	if _, exists := s.usersByEmail[email]; exists {
		return nil, ErrUserExists
	}

	s.nextUserID++
	u := &models.User{
		ID:           "u_" + strconv.Itoa(s.nextUserID),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	s.usersByID[u.ID] = u
	s.usersByEmail[email] = u
	return u, nil
}

// GetUserByEmail looks up a user by email (case-insensitive).
func (s *Store) GetUserByEmail(email string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	u, ok := s.usersByEmail[normalizeEmail(email)]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

// CreateTicket stores a new ticket owned by userID.
func (s *Store) CreateTicket(userID, title, description string) *models.Ticket {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextTicketID++
	now := time.Now()
	t := &models.Ticket{
		ID:          "t_" + strconv.Itoa(s.nextTicketID),
		UserID:      userID,
		Title:       title,
		Description: description,
		Status:      models.StatusOpen,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.tickets[t.ID] = t
	return t
}

// ListTicketsByUser returns all tickets owned by userID, newest first.
func (s *Store) ListTicketsByUser(userID string) []*models.Ticket {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*models.Ticket, 0)
	for _, t := range s.tickets {
		if t.UserID == userID {
			result = append(result, t)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

// GetTicket returns a ticket by ID regardless of owner; callers must perform
// their own ownership check so that "not found" vs "not yours" can be
// distinguished (404 vs 403) by the handler layer.
func (s *Store) GetTicket(id string) (*models.Ticket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tickets[id]
	if !ok {
		return nil, ErrTicketNotFound
	}
	return t, nil
}

// UpdateTicketStatus sets a new status on the ticket and bumps UpdatedAt.
func (s *Store) UpdateTicketStatus(id string, status models.TicketStatus) (*models.Ticket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tickets[id]
	if !ok {
		return nil, ErrTicketNotFound
	}
	t.Status = status
	t.UpdatedAt = time.Now()
	return t, nil
}
