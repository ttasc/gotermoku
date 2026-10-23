package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
)

const (
	hashIterations = 50000
	tokenBytesLen  = 32
	sessionTTL     = 24 * time.Hour
)

// ==========================================
// ANTI-SPAM: RATE LIMITER THEO IP
// ==========================================

type RateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	window   time.Duration
	maxHits  int
}

func NewRateLimiter(window time.Duration, maxHits int) *RateLimiter {
	rl := &RateLimiter{
		attempts: make(map[string][]time.Time),
		window:   window,
		maxHits:  maxHits,
	}

	// Luồng ngầm dọn rác IP cũ sau mỗi 5 phút để tránh rò rỉ RAM
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-rl.window)

	var valid []time.Time
	for _, t := range rl.attempts[ip] {
		if t.After(threshold) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= rl.maxHits {
		rl.attempts[ip] = valid
		return false
	}

	rl.attempts[ip] = append(valid, now)
	return true
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	threshold := now.Add(-rl.window)
	for ip, list := range rl.attempts {
		var valid []time.Time
		for _, t := range list {
			if t.After(threshold) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(rl.attempts, ip)
		} else {
			rl.attempts[ip] = valid
		}
	}
}

// ==========================================
// QUẢN LÝ PHIÊN (SESSION)
// ==========================================

type Session struct {
	Username  string
	ExpiresAt time.Time
}

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]Session),
	}
}

func (ss *SessionStore) Create(username string) (string, error) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	buf := make([]byte, tokenBytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := hex.EncodeToString(buf)

	ss.sessions[token] = Session{
		Username:  username,
		ExpiresAt: time.Now().Add(sessionTTL),
	}
	return token, nil
}

func (ss *SessionStore) Validate(token string) (string, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	sess, ok := ss.sessions[token]
	if !ok || time.Now().After(sess.ExpiresAt) {
		return "", false
	}
	return sess.Username, true
}

// ==========================================
// AUTH MANAGER
// ==========================================

type AuthManager struct {
	store   *UserStore
	limiter *RateLimiter
	session *SessionStore
}

func NewAuthManager(store *UserStore) *AuthManager {
	return &AuthManager{
		store:   store,
		limiter: NewRateLimiter(10*time.Second, 5), // Tối đa 5 yêu cầu / 10s trên 1 IP
		session: NewSessionStore(),
	}
}

func (a *AuthManager) Register(ip, username, password string) (string, error) {
	if !a.limiter.Allow(ip) {
		return "", errors.New("too many requests, please slow down")
	}

	username = strings.TrimSpace(username)
	if len(username) < 3 || len(username) > 20 {
		return "", errors.New("username must be between 3 and 20 characters")
	}
	if len(password) < 6 {
		return "", errors.New("password must be at least 6 characters")
	}

	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", err
	}
	salt := hex.EncodeToString(saltBytes)
	hash := computeHash(password, salt)

	user := &User{
		Username:     username,
		PasswordHash: hash,
		Salt:         salt,
		CreatedAt:    time.Now(),
	}

	if err := a.store.Create(user); err != nil {
		return "", err
	}

	return a.session.Create(username)
}

func (a *AuthManager) Login(ip, username, password string) (string, error) {
	if !a.limiter.Allow(ip) {
		return "", errors.New("too many requests, please slow down")
	}

	user, ok := a.store.Get(username)
	if !ok {
		return "", errors.New("invalid username or password")
	}

	expectedHash := computeHash(password, user.Salt)
	// So sánh thời gian hằng số (Constant Time Compare) để chống timing attack
	if subtle.ConstantTimeCompare([]byte(expectedHash), []byte(user.PasswordHash)) != 1 {
		return "", errors.New("invalid username or password")
	}

	return a.session.Create(username)
}

func (a *AuthManager) ValidateToken(token string) (string, bool) {
	return a.session.Validate(token)
}

func computeHash(password, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte(password))
	digest := h.Sum(nil)

	// Lặp 50,000 vòng SHA-256 để tăng chi phí tính toán cho kẻ tấn công
	for i := 0; i < hashIterations; i++ {
		h.Reset()
		h.Write(digest)
		digest = h.Sum(nil)
	}

	return hex.EncodeToString(digest)
}
