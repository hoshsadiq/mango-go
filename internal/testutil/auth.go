package testutil

import (
	"net/http"
	"testing"

	"github.com/vrsandeep/mango-go/internal/api"
	"github.com/vrsandeep/mango-go/internal/auth"
)

var precomputedTestPasswordHash = MustHashPassword("pw")

// MustHashPassword is like auth.HashPassword but panics on error.
func MustHashPassword(password string) string {
	hash, err := auth.HashPassword(password)
	if err != nil {
		panic("testutil: failed to hash password: " + err.Error())
	}
	return hash
}

// GetAuthCookie creates a user, creates a session, and returns a valid session cookie.
// Bypasses the login HTTP endpoint to avoid per-test request overhead.
func GetAuthCookie(t *testing.T, s *api.Server, username, password, role string) *http.Cookie {
	t.Helper()

	var passwordHash string
	if password == "pw" {
		passwordHash = precomputedTestPasswordHash
	} else {
		var err error
		passwordHash, err = auth.HashPassword(password)
		if err != nil {
			t.Fatalf("Failed to hash password for test user: %v", err)
		}
	}
	user, err := s.Store().CreateUser(username, passwordHash, role)
	if err != nil {
		t.Fatalf("Failed to create test user '%s': %v", username, err)
	}

	token, err := s.Store().CreateSession(user.ID)
	if err != nil {
		t.Fatalf("Failed to create session for test user '%s': %v", username, err)
	}
	return &http.Cookie{
		Name:     "session_token",
		Value:    token,
		HttpOnly: true,
		Path:     "/",
	}
}

func CookieForUser(t *testing.T, server *api.Server, username, password, role string) *http.Cookie {
	t.Helper()
	cookie := GetAuthCookie(t, server, username, password, role)
	if cookie == nil {
		t.Fatal("Failed to get session cookie after successful login for test user")
	}
	// on cleanup, delete the user
	t.Cleanup(func() {
		user, _ := server.Store().GetUserByUsername(username)
		server.Store().DeleteUser(user.ID)
	})
	return cookie
}
