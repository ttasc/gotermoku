package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go.etcd.io/bbolt"

	"github.com/ttasc/gotermoku/internal/common/core"
)

var usersBucket = []byte("users")

type User struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Salt         string    `json:"salt"`
	CreatedAt    time.Time `json:"created_at"`
	Elo          int       `json:"elo"`
	Wins         int       `json:"wins"`
	Losses       int       `json:"losses"`
	Draws        int       `json:"draws"`
}

type UserStore struct {
	db *bbolt.DB
}

// NewUserStore khởi tạo hoặc mở cơ sở dữ liệu bbolt với quyền hạn POSIX 0600
func NewUserStore(filePath string) (*UserStore, error) {
	db, err := bbolt.Open(filePath, 0600, &bbolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("failed to open bbolt database: %w", err)
	}

	// Đảm bảo bucket "users" luôn tồn tại
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(usersBucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize users bucket: %w", err)
	}

	return &UserStore{db: db}, nil
}

func (s *UserStore) Close() error {
	return s.db.Close()
}

func (s *UserStore) Get(username string) (*User, bool) {
	var user *User
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(usersBucket)
		if b == nil {
			return nil
		}
		data := b.Get([]byte(username))
		if data == nil {
			return nil
		}
		var u User
		if err := json.Unmarshal(data, &u); err != nil {
			return err
		}
		user = &u
		return nil
	})

	if err != nil || user == nil {
		return nil, false
	}
	return user, true
}

func (s *UserStore) Create(user *User) error {
	if user.Elo <= 0 {
		user.Elo = core.DefaultElo
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(usersBucket)
		k := []byte(user.Username)
		if b.Get(k) != nil {
			return errors.New("username already exists")
		}

		data, err := json.Marshal(user)
		if err != nil {
			return err
		}
		return b.Put(k, data)
	})
}

// UpdateStats cập nhật Elo và số trận thắng/thua/hòa trong 1 giao dịch ACID nguyên tử
func (s *UserStore) UpdateStats(username string, newElo int, won, lost, drew bool) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(usersBucket)
		k := []byte(username)
		data := b.Get(k)
		if data == nil {
			return errors.New("user not found")
		}

		var u User
		if err := json.Unmarshal(data, &u); err != nil {
			return err
		}

		u.Elo = newElo
		if won {
			u.Wins++
		} else if lost {
			u.Losses++
		} else if drew {
			u.Draws++
		}

		updatedData, err := json.Marshal(&u)
		if err != nil {
			return err
		}
		return b.Put(k, updatedData)
	})
}
