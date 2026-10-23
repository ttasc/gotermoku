package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type User struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Salt         string    `json:"salt"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserStore struct {
	mu       sync.RWMutex
	filePath string
	users    map[string]*User
}

// NewUserStore khởi tạo bộ nhớ người dùng từ file JSON phẳng
func NewUserStore(filePath string) (*UserStore, error) {
	store := &UserStore{
		filePath: filePath,
		users:    make(map[string]*User),
	}

	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *UserStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if os.IsNotExist(err) {
		return nil // File chưa tồn tại -> bắt đầu với map rỗng
	}
	if err != nil {
		return fmt.Errorf("reading user db: %w", err)
	}

	if len(data) == 0 {
		return nil
	}

	var list []*User
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("unmarshaling users: %w", err)
	}

	for _, u := range list {
		s.users[u.Username] = u
	}
	return nil
}

// save thực hiện ghi nguyên tử (Atomic write) chuẩn POSIX thông qua temp file + os.Rename
func (s *UserStore) save() error {
	var list []*User
	for _, u := range s.users {
		list = append(list, u)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmpFile := fmt.Sprintf("%s.tmp.%d", s.filePath, time.Now().UnixNano())
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return err
	}

	// os.Rename trong POSIX là thao tác atomic, đảm bảo file không bao giờ bị hỏng giữa chừng
	if err := os.Rename(tmpFile, s.filePath); err != nil {
		os.Remove(tmpFile)
		return err
	}

	return nil
}

func (s *UserStore) Get(username string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, exists := s.users[username]
	return user, exists
}

func (s *UserStore) Create(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.Username]; exists {
		return errors.New("username already exists")
	}

	s.users[user.Username] = user
	return s.save()
}
