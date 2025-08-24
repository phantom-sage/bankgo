package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/o1egl/paseto/v2"
	"golang.org/x/crypto/bcrypt"

	"github.com/phantom-sage/bankgo/internal/admin/interfaces"
)

// AdminAuthServiceImpl implements the AdminAuthService interface
type AdminAuthServiceImpl struct {
	secretKey        []byte
	sessionTimeout   time.Duration
	defaultUsername  string
	defaultPassword  string
	currentUsername  string
	currentPassword  string
	activeSessions   map[string]*interfaces.AdminSession
	sessionMutex     sync.RWMutex
	credentialsMutex sync.RWMutex
}

// AdminTokenClaims represents the claims in an admin PASETO token
type AdminTokenClaims struct {
	SessionID string    `json:"session_id"`
	Username  string    `json:"username"`
	IssuedAt  time.Time `json:"iat"`
	ExpiresAt time.Time `json:"exp"`
}

// NewAdminAuthService creates a new admin authentication service
func NewAdminAuthService(secretKey string, sessionTimeout time.Duration, defaultUsername, defaultPassword string) (interfaces.AdminAuthService, error) {
	if len(secretKey) < 32 {
		return nil, errors.New("secret key must be at least 32 characters long")
	}

	// PASETO v2 requires exactly 32 bytes for the key
	key := []byte(secretKey)
	if len(key) > 32 {
		key = key[:32] // Truncate to 32 bytes
	} else if len(key) < 32 {
		// Pad with zeros if less than 32 bytes
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	}

	// Hash the default password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash default password: %w", err)
	}

	return &AdminAuthServiceImpl{
		secretKey:        key,
		sessionTimeout:   sessionTimeout,
		defaultUsername:  defaultUsername,
		defaultPassword:  defaultPassword,
		currentUsername:  defaultUsername,
		currentPassword:  string(hashedPassword),
		activeSessions:   make(map[string]*interfaces.AdminSession),
		sessionMutex:     sync.RWMutex{},
		credentialsMutex: sync.RWMutex{},
	}, nil
}

// Login authenticates admin credentials and returns a session token
func (s *AdminAuthServiceImpl) Login(ctx context.Context, username, password string) (*interfaces.AdminSession, error) {
	if username == "" || password == "" {
		return nil, errors.New("username and password are required")
	}

	s.credentialsMutex.RLock()
	currentUsername := s.currentUsername
	currentPassword := s.currentPassword
	s.credentialsMutex.RUnlock()

	// Validate credentials
	if username != currentUsername {
		return nil, errors.New("invalid credentials")
	}

	// Check password
	if err := bcrypt.CompareHashAndPassword([]byte(currentPassword), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate session ID
	sessionID, err := s.generateSessionID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	// Create token claims
	now := time.Now()
	claims := AdminTokenClaims{
		SessionID: sessionID,
		Username:  username,
		IssuedAt:  now,
		ExpiresAt: now.Add(s.sessionTimeout),
	}

	// Generate PASETO token
	token, err := paseto.NewV2().Encrypt(s.secretKey, claims, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Create session
	session := &interfaces.AdminSession{
		ID:          sessionID,
		Username:    username,
		PasetoToken: token,
		ExpiresAt:   claims.ExpiresAt,
		CreatedAt:   now,
		LastActive:  now,
	}

	// Store session
	s.sessionMutex.Lock()
	s.activeSessions[sessionID] = session
	s.sessionMutex.Unlock()

	return session, nil
}

// ValidateSession validates a PASETO token and returns session info
func (s *AdminAuthServiceImpl) ValidateSession(ctx context.Context, token string) (*interfaces.AdminSession, error) {
	if token == "" {
		return nil, errors.New("token cannot be empty")
	}

	// Decrypt and validate token
	var claims AdminTokenClaims
	err := paseto.NewV2().Decrypt(token, s.secretKey, &claims, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Check if token is expired
	if time.Now().After(claims.ExpiresAt) {
		// Remove expired session
		s.sessionMutex.Lock()
		delete(s.activeSessions, claims.SessionID)
		s.sessionMutex.Unlock()
		return nil, errors.New("token has expired")
	}

	// Validate claims
	if claims.SessionID == "" {
		return nil, errors.New("invalid session ID in token")
	}

	if claims.Username == "" {
		return nil, errors.New("invalid username in token")
	}

	// Check if session exists and is active
	s.sessionMutex.RLock()
	session, exists := s.activeSessions[claims.SessionID]
	s.sessionMutex.RUnlock()

	if !exists {
		return nil, errors.New("session not found or has been invalidated")
	}

	// Update last active time
	now := time.Now()
	s.sessionMutex.Lock()
	session.LastActive = now
	s.sessionMutex.Unlock()

	return session, nil
}

// RefreshSession extends the session expiration
func (s *AdminAuthServiceImpl) RefreshSession(ctx context.Context, token string) (*interfaces.AdminSession, error) {
	// First validate the current session
	session, err := s.ValidateSession(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("cannot refresh invalid session: %w", err)
	}

	// Generate new token with extended expiration
	now := time.Now()
	claims := AdminTokenClaims{
		SessionID: session.ID,
		Username:  session.Username,
		IssuedAt:  now,
		ExpiresAt: now.Add(s.sessionTimeout),
	}

	// Generate new PASETO token
	newToken, err := paseto.NewV2().Encrypt(s.secretKey, claims, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refreshed token: %w", err)
	}

	// Update session
	s.sessionMutex.Lock()
	session.PasetoToken = newToken
	session.ExpiresAt = claims.ExpiresAt
	session.LastActive = now
	s.sessionMutex.Unlock()

	return session, nil
}

// Logout invalidates a session
func (s *AdminAuthServiceImpl) Logout(ctx context.Context, token string) error {
	if token == "" {
		return errors.New("token cannot be empty")
	}

	// Decrypt token to get session ID
	var claims AdminTokenClaims
	err := paseto.NewV2().Decrypt(token, s.secretKey, &claims, nil)
	if err != nil {
		// If we can't decrypt the token, consider it already invalid
		return nil
	}

	// Remove session from active sessions
	s.sessionMutex.Lock()
	delete(s.activeSessions, claims.SessionID)
	s.sessionMutex.Unlock()

	return nil
}

// UpdateCredentials changes admin credentials
func (s *AdminAuthServiceImpl) UpdateCredentials(ctx context.Context, username, oldPassword, newPassword string) error {
	if username == "" || oldPassword == "" || newPassword == "" {
		return errors.New("username, old password, and new password are required")
	}

	if len(newPassword) < 8 {
		return errors.New("new password must be at least 8 characters long")
	}

	s.credentialsMutex.Lock()
	defer s.credentialsMutex.Unlock()

	// Validate current credentials
	if username != s.currentUsername {
		return errors.New("invalid username")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(s.currentPassword), []byte(oldPassword)); err != nil {
		return errors.New("invalid old password")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update credentials
	s.currentPassword = string(hashedPassword)

	// Invalidate all active sessions to force re-authentication
	s.sessionMutex.Lock()
	s.activeSessions = make(map[string]*interfaces.AdminSession)
	s.sessionMutex.Unlock()

	return nil
}

// generateSessionID generates a cryptographically secure session ID
func (s *AdminAuthServiceImpl) generateSessionID() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetActiveSessionCount returns the number of active sessions (for monitoring)
func (s *AdminAuthServiceImpl) GetActiveSessionCount() int {
	s.sessionMutex.RLock()
	defer s.sessionMutex.RUnlock()
	return len(s.activeSessions)
}

// CleanupExpiredSessions removes expired sessions from memory
func (s *AdminAuthServiceImpl) CleanupExpiredSessions() {
	now := time.Now()
	s.sessionMutex.Lock()
	defer s.sessionMutex.Unlock()

	for sessionID, session := range s.activeSessions {
		if now.After(session.ExpiresAt) {
			delete(s.activeSessions, sessionID)
		}
	}
}

// IsDefaultCredentials returns true if still using default credentials
func (s *AdminAuthServiceImpl) IsDefaultCredentials() bool {
	s.credentialsMutex.RLock()
	defer s.credentialsMutex.RUnlock()
	
	// Check if current username is still the default
	if s.currentUsername != s.defaultUsername {
		return false
	}
	
	// Check if current password is still the default (compare hashes)
	err := bcrypt.CompareHashAndPassword([]byte(s.currentPassword), []byte(s.defaultPassword))
	return err == nil
}