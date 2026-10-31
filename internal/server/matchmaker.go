package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ttasc/gotermoku/internal/common/core"
)

const (
	roomCodeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	roomCodeLength  = 6
)

type MatchResult struct {
	MatchID       string
	OpponentName  string
	AssignedColor uint8 // core.White hoặc core.Black
	Rows          int
	Cols          int
	OpponentElo   int
	YourElo       int
	RoomCode      string
}

type QueueEntry struct {
	Username   string
	Token      string
	Elo        int
	EnqueuedAt time.Time
	ResultCh   chan MatchResult
}

type PrivateRoom struct {
	Code      string
	Rows      int
	Cols      int
	HostEntry *QueueEntry
	CreatedAt time.Time
}

type Matchmaker struct {
	mu           sync.Mutex
	queue        []*QueueEntry
	privateRooms map[string]*PrivateRoom
	roomMgr      *RoomManager
	authMgr      *AuthManager
	defaultRows  int
	defaultCols  int
}

func NewMatchmaker(roomMgr *RoomManager, authMgr *AuthManager, rows, cols int) *Matchmaker {
	m := &Matchmaker{
		queue:        make([]*QueueEntry, 0),
		privateRooms: make(map[string]*PrivateRoom),
		roomMgr:      roomMgr,
		authMgr:      authMgr,
		defaultRows:  rows,
		defaultCols:  cols,
	}

	// Luồng quét ghép trận Elo định kỳ
	go m.matchmakingLoop()
	// Luồng dọn dẹp phòng riêng hết hạn
	go m.cleanupLoop()

	return m
}

// Enqueue đưa người chơi vào hàng đợi Elo và cố gắng ghép trận
func (m *Matchmaker) Enqueue(username, token string) (<-chan MatchResult, func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	elo := core.DefaultElo
	if user, ok := m.authMgr.GetUser(username); ok && user.Elo > 0 {
		elo = user.Elo
	}

	resultCh := make(chan MatchResult, 1)
	entry := &QueueEntry{
		Username:   username,
		Token:      token,
		Elo:        elo,
		EnqueuedAt: time.Now(),
		ResultCh:   resultCh,
	}

	m.queue = append(m.queue, entry)

	// Thử ghép ngay nếu có người trong dải Elo phù hợp
	m.tryMatchQueueLocked()

	return resultCh, m.makeCancelFunc(entry)
}

func (m *Matchmaker) matchmakingLoop() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		m.mu.Lock()
		m.tryMatchQueueLocked()
		m.mu.Unlock()
	}
}

func (m *Matchmaker) tryMatchQueueLocked() {
	if len(m.queue) < 2 {
		return
	}

	now := time.Now()
	matched := make(map[int]bool)

	for i := 0; i < len(m.queue); i++ {
		if matched[i] {
			continue
		}
		e1 := m.queue[i]
		waitSec1 := int(now.Sub(e1.EnqueuedAt).Seconds())
		// Nới rộng dải Elo: 100 điểm ban đầu + 50 điểm mỗi 5 giây chờ
		allowedDelta1 := 100 + (waitSec1/5)*50

		bestJ := -1
		bestDiff := 1000000

		for j := i + 1; j < len(m.queue); j++ {
			if matched[j] {
				continue
			}
			e2 := m.queue[j]
			if e1.Username == e2.Username {
				continue
			}

			diff := abs(e1.Elo - e2.Elo)
			waitSec2 := int(now.Sub(e2.EnqueuedAt).Seconds())
			allowedDelta2 := 100 + (waitSec2/5)*50
			maxAllowed := max(allowedDelta1, allowedDelta2)

			if diff <= maxAllowed && diff < bestDiff {
				bestDiff = diff
				bestJ = j
			}
		}

		if bestJ != -1 {
			matched[i] = true
			matched[bestJ] = true
			m.createMatchLocked(e1, m.queue[bestJ], "", m.defaultRows, m.defaultCols)
		}
	}

	if len(matched) > 0 {
		newQueue := make([]*QueueEntry, 0, len(m.queue)-len(matched))
		for i, entry := range m.queue {
			if !matched[i] {
				newQueue = append(newQueue, entry)
			}
		}
		m.queue = newQueue
	}
}

// ==========================================
// PRIVATE ROOMS (PHÒNG RIÊNG)
// ==========================================

func (m *Matchmaker) CreatePrivateRoom(rows, cols int) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if rows < 3 || cols < 3 {
		rows = m.defaultRows
		cols = m.defaultCols
	}

	for i := 0; i < 10; i++ {
		code := generateRoomCode()
		if _, exists := m.privateRooms[code]; !exists {
			m.privateRooms[code] = &PrivateRoom{
				Code:      code,
				Rows:      rows,
				Cols:      cols,
				CreatedAt: time.Now(),
			}
			return code, nil
		}
	}
	return "", errors.New("failed to generate unique room code")
}

func (m *Matchmaker) JoinPrivateRoom(code, username, token string) (<-chan MatchResult, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, exists := m.privateRooms[code]
	if !exists {
		return nil, nil, errors.New("private room not found or expired")
	}

	elo := core.DefaultElo
	if user, ok := m.authMgr.GetUser(username); ok && user.Elo > 0 {
		elo = user.Elo
	}

	resultCh := make(chan MatchResult, 1)
	entry := &QueueEntry{
		Username:   username,
		Token:      token,
		Elo:        elo,
		EnqueuedAt: time.Now(),
		ResultCh:   resultCh,
	}

	// Người thứ nhất (Host) vào phòng chờ
	if room.HostEntry == nil {
		room.HostEntry = entry
		cancel := func() {
			m.mu.Lock()
			defer m.mu.Unlock()
			if r, ok := m.privateRooms[code]; ok && r.HostEntry == entry {
				delete(m.privateRooms, code)
				close(entry.ResultCh)
			}
		}
		return resultCh, cancel, nil
	}

	// Không cho phép tự đấu với chính mình
	if room.HostEntry.Username == username {
		return nil, nil, errors.New("cannot join your own private room twice")
	}

	// Người thứ hai vào -> Ghép trận và giải phóng mã phòng
	hostEntry := room.HostEntry
	delete(m.privateRooms, code)

	m.createMatchLocked(hostEntry, entry, code, room.Rows, room.Cols)
	return resultCh, func() {}, nil
}

func (m *Matchmaker) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		m.mu.Lock()
		threshold := time.Now().Add(-15 * time.Minute)
		for code, pr := range m.privateRooms {
			if pr.CreatedAt.Before(threshold) {
				if pr.HostEntry != nil {
					close(pr.HostEntry.ResultCh)
				}
				delete(m.privateRooms, code)
			}
		}
		m.mu.Unlock()
	}
}

// ==========================================
// TẠO PHÒNG ĐẤU VÀ PHÁT TÍN HIỆU
// ==========================================

func (m *Matchmaker) createMatchLocked(e1, e2 *QueueEntry, roomCode string, rows, cols int) {
	matchID := generateMatchID()
	p1IsWhite := randomBool()

	var whiteEntry, blackEntry *QueueEntry
	if p1IsWhite {
		whiteEntry, blackEntry = e1, e2
	} else {
		whiteEntry, blackEntry = e2, e1
	}

	room := NewRoom(
		matchID,
		whiteEntry.Username, whiteEntry.Token,
		blackEntry.Username, blackEntry.Token,
		rows, cols,
		m.roomMgr.Remove,
		m.authMgr, // Bổ sung AuthManager
	)
	m.roomMgr.Add(room)

	whiteEntry.ResultCh <- MatchResult{
		MatchID:       matchID,
		OpponentName:  blackEntry.Username,
		AssignedColor: core.White,
		Rows:          rows,
		Cols:          cols,
		OpponentElo:   blackEntry.Elo,
		YourElo:       whiteEntry.Elo,
		RoomCode:      roomCode,
	}

	blackEntry.ResultCh <- MatchResult{
		MatchID:       matchID,
		OpponentName:  whiteEntry.Username,
		AssignedColor: core.Black,
		Rows:          rows,
		Cols:          cols,
		OpponentElo:   whiteEntry.Elo,
		YourElo:       blackEntry.Elo,
		RoomCode:      roomCode,
	}
}

func (m *Matchmaker) makeCancelFunc(target *QueueEntry) func() {
	return func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		for i, entry := range m.queue {
			if entry == target {
				m.queue = append(m.queue[:i], m.queue[i+1:]...)
				close(entry.ResultCh)
				break
			}
		}
	}
}

func generateMatchID() string {
	bytes := make([]byte, 8)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func generateRoomCode() string {
	bytes := make([]byte, roomCodeLength)
	_, _ = rand.Read(bytes)
	for i := 0; i < roomCodeLength; i++ {
		bytes[i] = roomCodeCharset[int(bytes[i])%len(roomCodeCharset)]
	}
	return string(bytes)
}

func randomBool() bool {
	n, err := rand.Int(rand.Reader, big.NewInt(2))
	if err != nil {
		return true
	}
	return n.Int64() == 0
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
