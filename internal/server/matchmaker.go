package server

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"sync"

	"github.com/ttasc/gotermoku/internal/core"
)

type MatchResult struct {
	MatchID       string
	OpponentName  string
	AssignedColor uint8 // core.White hoặc core.Black
	Rows          int
	Cols          int
}

type QueueEntry struct {
	Username string
	Token    string
	ResultCh chan MatchResult
}

type Matchmaker struct {
	mu          sync.Mutex
	queue       []*QueueEntry
	roomMgr     *RoomManager
	defaultRows int
	defaultCols int
}

func NewMatchmaker(roomMgr *RoomManager, rows, cols int) *Matchmaker {
	return &Matchmaker{
		queue:       make([]*QueueEntry, 0),
		roomMgr:     roomMgr,
		defaultRows: rows,
		defaultCols: cols,
	}
}

// Enqueue đưa người chơi vào hàng đợi và trả về kênh nhận kết quả ghép cùng hàm huỷ (cancel)
func (m *Matchmaker) Enqueue(username, token string) (<-chan MatchResult, func()) {
	m.mu.Lock()
	defer m.mu.Unlock()

	resultCh := make(chan MatchResult, 1)
	entry := &QueueEntry{
		Username: username,
		Token:    token,
		ResultCh: resultCh,
	}

	// Nếu đã có sẵn người chơi đang chờ, tiến hành ghép cặp ngay lập tức
	if len(m.queue) > 0 {
		opponent := m.queue[0]
		m.queue = m.queue[1:]

		// Không cho phép tự ghép với chính mình nếu cùng 1 tài khoản mở 2 terminal
		if opponent.Username == username {
			m.queue = append(m.queue, entry)
			return resultCh, m.makeCancelFunc(entry)
		}

		matchID := generateMatchID()

		// Tung đồng xu ngẫu nhiên phân định Trắng/Đen
		p1IsWhite := randomBool()

		var whiteEntry, blackEntry *QueueEntry
		if p1IsWhite {
			whiteEntry, blackEntry = entry, opponent
		} else {
			whiteEntry, blackEntry = opponent, entry
		}

		// Tạo phòng đấu mới trên Server
		room := NewRoom(
			matchID,
			whiteEntry.Username, whiteEntry.Token,
			blackEntry.Username, blackEntry.Token,
			m.defaultRows, m.defaultCols,
			m.roomMgr.Remove,
		)
		m.roomMgr.Add(room)

		// Thông báo kết quả cho cả 2 người chơi
		whiteEntry.ResultCh <- MatchResult{
			MatchID:       matchID,
			OpponentName:  blackEntry.Username,
			AssignedColor: core.White,
			Rows:          m.defaultRows,
			Cols:          m.defaultCols,
		}

		blackEntry.ResultCh <- MatchResult{
			MatchID:       matchID,
			OpponentName:  whiteEntry.Username,
			AssignedColor: core.Black,
			Rows:          m.defaultRows,
			Cols:          m.defaultCols,
		}

		return resultCh, func() {}
	}

	// Nếu chưa có ai, xếp vào hàng đợi FIFO
	m.queue = append(m.queue, entry)
	return resultCh, m.makeCancelFunc(entry)
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
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func randomBool() bool {
	n, err := rand.Int(rand.Reader, big.NewInt(2))
	if err != nil {
		return true
	}
	return n.Int64() == 0
}
