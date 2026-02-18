// Package auth provides session management, password validation, and brute-force protection.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/adcondev/ticket-daemon/internal/config"
)

const (
	SessionCookieName = "td_session"
	SessionDuration   = 15 * time.Minute
	MaxLoginAttempts  = 5
	LockoutDuration   = 5 * time.Minute
	CleanupInterval   = 5 * time.Minute
)

type failInfo struct {
	count       int
	lockedUntil time.Time
}

// Manager handles session lifecycle, password validation, and login throttling.
type Manager struct {
	sessions     map[string]time.Time
	failedLogins map[string]failInfo
	mu           sync.RWMutex
}

// NewManager creates an auth manager with a cleanup goroutine bound to ctx.
func NewManager(ctx context.Context) *Manager {
	m := &Manager{
		sessions:     make(map[string]time.Time),
		failedLogins: make(map[string]failInfo),
	}
	go m.cleanupLoop(ctx)
	log.Printf("[i] Auth manager initialized (enabled=%v)", m.Enabled())
	return m
}

// Enabled returns true if a password hash was injected at build time.
func (m *Manager) Enabled() bool {
	return config.PasswordHashB64 != ""
}

// ValidatePassword decodes the base64 hash and compares with bcrypt.
func (m *Manager) ValidatePassword(password string) bool {
	if !m.Enabled() {
		log.Println("[!] AUTH DISABLED: No password hash configured (dev mode)")
		return true
	}
	hashBytes, err := base64.StdEncoding.DecodeString(config.PasswordHashB64)
	if err != nil {
		log.Printf("[X] Failed to decode password hash from base64: %v", err)
		return false
	}
	return bcrypt.CompareHashAndPassword(hashBytes, []byte(password)) == nil
}

// CreateSession generates a cryptographically random session token.
func (m *Manager) CreateSession() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Printf("[!] crypto/rand failed: %v", err)
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	token := hex.EncodeToString(b)
	m.mu.Lock()
	m.sessions[token] = time.Now().Add(SessionDuration)
	m.mu.Unlock()
	return token
}

// ValidateSession checks if a token exists and has not expired.
func (m *Manager) ValidateSession(token string) bool {
	if token == "" {
		return false
	}
	m.mu.RLock()
	expiry, exists := m.sessions[token]
	m.mu.RUnlock()
	if !exists {
		return false
	}
	if time.Now().After(expiry) {
		m.mu.Lock()
		delete(m.sessions, token)
		m.mu.Unlock()
		return false
	}
	return true
}

// IsLockedOut returns true if the IP has exceeded MaxLoginAttempts.
func (m *Manager) IsLockedOut(ip string) bool {
	m.mu.RLock()
	info, exists := m.failedLogins[ip]
	m.mu.RUnlock()
	if !exists {
		return false
	}
	return info.count >= MaxLoginAttempts && time.Now().Before(info.lockedUntil)
}

// RecordFailedLogin increments the failure counter for an IP.
func (m *Manager) RecordFailedLogin(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	info := m.failedLogins[ip]
	info.count++
	if info.count >= MaxLoginAttempts {
		info.lockedUntil = time.Now().Add(LockoutDuration)
		log.Printf("[AUDIT] IP %s locked out for %v after %d failed attempts",
			ip, LockoutDuration, info.count)
	}
	m.failedLogins[ip] = info
}

// ClearFailedLogins resets the counter on successful login.
func (m *Manager) ClearFailedLogins(ip string) {
	m.mu.Lock()
	delete(m.failedLogins, ip)
	m.mu.Unlock()
}

// SetSessionCookie writes a secure, HttpOnly session cookie.
func (m *Manager) SetSessionCookie(w http.ResponseWriter) string {
	token := m.CreateSession()
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(SessionDuration.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	return token
}

// ClearSessionCookie removes the session cookie.
func (m *Manager) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// GetSessionFromRequest extracts and validates the session from cookies.
func (m *Manager) GetSessionFromRequest(r *http.Request) bool {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return false
	}
	return m.ValidateSession(cookie.Value)
}

func (m *Manager) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(CleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Println("[i] Auth cleanup goroutine stopped")
			return
		case <-ticker.C:
			m.mu.Lock()
			now := time.Now()
			for k, v := range m.sessions {
				if now.After(v) {
					delete(m.sessions, k)
				}
			}
			for k, v := range m.failedLogins {
				if v.count >= MaxLoginAttempts && now.After(v.lockedUntil) {
					delete(m.failedLogins, k)
				}
			}
			m.mu.Unlock()
		}
	}
}
