// SPDX-License-Identifier: AGPL-3.0-or-later

package ideachat

import (
	"testing"
	"time"
)

func TestSessionCreateAndGet(t *testing.T) {
	store := NewStore()
	sess := store.Create("myproject", "user@example.com")
	if sess == nil {
		t.Fatal("Create returned nil")
	}
	if sess.ID == "" {
		t.Fatal("session ID is empty")
	}
	if sess.ProjectSlug != "myproject" {
		t.Errorf("ProjectSlug = %q, want %q", sess.ProjectSlug, "myproject")
	}
	if sess.UserEmail != "user@example.com" {
		t.Errorf("UserEmail = %q, want %q", sess.UserEmail, "user@example.com")
	}
	if sess.Status != StatusConversing {
		t.Errorf("Status = %q, want %q", sess.Status, StatusConversing)
	}

	got, ok := store.Get(sess.ID)
	if !ok {
		t.Fatal("Get returned false for a known session")
	}
	if got.ID != sess.ID {
		t.Errorf("Get returned wrong session")
	}
}

func TestSessionGetUnknown(t *testing.T) {
	store := NewStore()
	_, ok := store.Get("does-not-exist")
	if ok {
		t.Error("expected false for unknown session ID")
	}
}

func TestSessionTouch(t *testing.T) {
	store := NewStore()
	sess := store.Create("p", "u@e.com")
	old := sess.LastActivity

	time.Sleep(2 * time.Millisecond)
	store.Touch(sess.ID)

	got, _ := store.Get(sess.ID)
	if !got.LastActivity.After(old) {
		t.Error("Touch did not update LastActivity")
	}
}

func TestSessionDelete(t *testing.T) {
	store := NewStore()
	sess := store.Create("p", "u@e.com")
	store.Delete(sess.ID)
	_, ok := store.Get(sess.ID)
	if ok {
		t.Error("expected false after Delete")
	}
}

func TestSessionExpiry(t *testing.T) {
	store := NewStore()
	sess := store.Create("p", "u@e.com")
	// Force expiry by back-dating LastActivity.
	store.mu.Lock()
	store.sessions[sess.ID].LastActivity = time.Now().Add(-31 * time.Minute)
	store.mu.Unlock()

	_, ok := store.Get(sess.ID)
	if ok {
		t.Error("expected Get to return false for expired session")
	}
}
