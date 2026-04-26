// Package ideachat implements the conversational idea-capture session store
// and LLM conversation logic.
package ideachat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"github.com/kaos-control/kaos-control/internal/artifact"
)

const (
	sessionTTL     = 30 * time.Minute
	reaperInterval = 5 * time.Minute

	StatusConversing = "conversing"
	StatusProposed   = "proposed"
	StatusCreated    = "created"
)

// Message is one turn in the conversation.
type Message struct {
	Role    string `json:"role"`    // "user" | "assistant"
	Content string `json:"content"`
}

// Session holds conversational state for a single idea-capture session.
type Session struct {
	ID           string
	ProjectSlug  string
	UserEmail    string
	Messages     []Message
	Status       string // StatusConversing | StatusProposed | StatusCreated
	ClarifyCount int
	ProposedFM   artifact.Frontmatter
	ProposedBody string
	ProposedSlug string
	CreatedAt    time.Time
	LastActivity time.Time
}

// Store is a thread-safe in-memory session store.
type Store struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewStore returns an initialised Store.
func NewStore() *Store {
	return &Store{sessions: make(map[string]*Session)}
}

// Create allocates a new session with a crypto-random ID.
func (s *Store) Create(projectSlug, userEmail string) *Session {
	id := randomHex(16)
	now := time.Now()
	sess := &Session{
		ID:           id,
		ProjectSlug:  projectSlug,
		UserEmail:    userEmail,
		Status:       StatusConversing,
		CreatedAt:    now,
		LastActivity: now,
	}
	s.mu.Lock()
	s.sessions[id] = sess
	s.mu.Unlock()
	return sess
}

// Get returns the session for id, or nil/false if unknown or expired.
func (s *Store) Get(id string) (*Session, bool) {
	s.mu.RLock()
	sess, ok := s.sessions[id]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Since(sess.LastActivity) > sessionTTL {
		s.Delete(id)
		return nil, false
	}
	return sess, true
}

// Touch updates the last-activity timestamp for id.
func (s *Store) Touch(id string) {
	s.mu.Lock()
	if sess, ok := s.sessions[id]; ok {
		sess.LastActivity = time.Now()
	}
	s.mu.Unlock()
}

// Delete removes the session with id from the store.
func (s *Store) Delete(id string) {
	s.mu.Lock()
	delete(s.sessions, id)
	s.mu.Unlock()
}

// StartReaper launches a background goroutine that purges expired sessions
// every reaperInterval until ctx is cancelled.
func (s *Store) StartReaper(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(reaperInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.reap()
			}
		}
	}()
}

func (s *Store) reap() {
	now := time.Now()
	s.mu.Lock()
	for id, sess := range s.sessions {
		if now.Sub(sess.LastActivity) > sessionTTL {
			delete(s.sessions, id)
		}
	}
	s.mu.Unlock()
}

// randomHex generates n random bytes and returns them as a lowercase hex string.
func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
