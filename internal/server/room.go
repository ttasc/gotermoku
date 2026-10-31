package server

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ttasc/gotermoku/internal/common/core"
	"github.com/ttasc/gotermoku/internal/common/pb"
)

type RoomStatus int

const (
	RoomActive RoomStatus = iota
	RoomPaused
	RoomClosed
)

const defaultGraceSeconds = 45

type RoomPlayer struct {
	Username  string
	Token     string
	Color     uint8
	Elo       int
	Connected bool
	SendChan  chan *pb.GameSync
}

type Room struct {
	ID             string
	mu             sync.Mutex
	State          *core.GameState
	Status         RoomStatus
	White          *RoomPlayer
	Black          *RoomPlayer
	onRemove       func(roomID string)
	authMgr        *AuthManager
	eloUpdated     bool
	graceStopChan  chan struct{}
	graceRemaining int
}

func NewRoom(id string, whiteUser, whiteToken, blackUser, blackToken string, rows, cols int, onRemove func(string), authMgr *AuthManager) *Room {
	whiteElo := core.DefaultElo
	blackElo := core.DefaultElo

	if authMgr != nil {
		if u, ok := authMgr.GetUser(whiteUser); ok && u.Elo > 0 {
			whiteElo = u.Elo
		}
		if u, ok := authMgr.GetUser(blackUser); ok && u.Elo > 0 {
			blackElo = u.Elo
		}
	}

	return &Room{
		ID:     id,
		State:  core.NewGameState(rows, cols),
		Status: RoomActive,
		White: &RoomPlayer{
			Username:  whiteUser,
			Token:     whiteToken,
			Color:     core.White,
			Elo:       whiteElo,
			Connected: true,
			SendChan:  make(chan *pb.GameSync, 10),
		},
		Black: &RoomPlayer{
			Username:  blackUser,
			Token:     blackToken,
			Color:     core.Black,
			Elo:       blackElo,
			Connected: true,
			SendChan:  make(chan *pb.GameSync, 10),
		},
		onRemove: onRemove,
		authMgr:  authMgr,
	}
}

// SyncToPlayer gửi toàn bộ trạng thái hiện tại riêng cho 1 người chơi
func (r *Room) SyncToPlayer(token string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	player := r.getPlayerByToken(token)
	if player == nil {
		return
	}

	syncMsg := r.toGameSyncLocked("sync", "")
	select {
	case player.SendChan <- syncMsg:
	default:
	}
}

func (r *Room) HandleAction(token string, action *pb.ClientAction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Status != RoomActive {
		return errors.New("game is currently paused or closed")
	}

	player := r.getPlayerByToken(token)
	if player == nil {
		return errors.New("unauthorized action: player not in this room")
	}

	switch action.Type {
	case "move":
		if r.State.CurrentTurn != player.Color {
			return errors.New("not your turn")
		}
		x, y := int(action.X), int(action.Y)
		if !core.CanPlacePiece(r.State, x, y) {
			return errors.New("invalid move coordinate")
		}
		core.PlacePiece(r.State, x, y, player.Color)

		// Kiểm tra kết thúc ván (Thắng hoặc Hòa)
		if r.State.Winner != core.Empty {
			r.updateEloOnGameEndLocked()
		}

		r.broadcastLocked("sync", "")

	case "restart":
		if r.State.Winner != core.Empty {
			r.State.Reset()
			r.eloUpdated = false
			r.broadcastLocked("sync", "")
		}
	}

	return nil
}

// HandleDisconnect xử lý ngắt kết nối với thời gian ân hạn 45 giây
func (r *Room) HandleDisconnect(token string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Status == RoomClosed {
		return
	}

	leaver := r.getPlayerByToken(token)
	if leaver == nil {
		return
	}
	leaver.Connected = false

	// Nếu ván đấu đã kết thúc từ trước, đóng phòng luôn
	if r.State.Winner != core.Empty {
		r.closeRoomLocked()
		return
	}

	other := r.getOtherPlayer(token)
	// Nếu cả 2 đều đã ngắt kết nối, hủy phòng ngay lập tức
	if other == nil || !other.Connected {
		r.closeRoomLocked()
		return
	}

	// Chuyển sang trạng thái tạm dừng và bắt đầu đếm ngược ân hạn
	r.Status = RoomPaused
	r.graceRemaining = defaultGraceSeconds
	r.graceStopChan = make(chan struct{})

	msg := fmt.Sprintf("%s disconnected. Waiting for reconnect (%ds)...", leaver.Username, r.graceRemaining)
	r.broadcastLocked("pause", msg)

	go r.runGracePeriodCountdown(r.graceStopChan, token)
}

func (r *Room) runGracePeriodCountdown(stopChan chan struct{}, leaverToken string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			r.mu.Lock()
			if r.Status != RoomPaused {
				r.mu.Unlock()
				return
			}

			r.graceRemaining--

			if r.graceRemaining <= 0 {
				// Hết thời gian ân hạn -> Xử thua do bỏ cuộc (Forfeit)
				other := r.getOtherPlayer(leaverToken)
				leaver := r.getPlayerByToken(leaverToken)

				if other != nil && leaver != nil {
					r.State.Winner = other.Color
					r.updateEloOnGameEndLocked()
					forfeitMsg := fmt.Sprintf("%s forfeited due to disconnection timeout. %s wins!", leaver.Username, other.Username)
					r.broadcastLocked("disconnect", forfeitMsg)
				}

				r.closeRoomLocked()
				r.mu.Unlock()
				return
			}

			// Cập nhật số giây còn lại cho đối thủ
			if r.graceRemaining%5 == 0 || r.graceRemaining <= 5 {
				msg := fmt.Sprintf("Waiting for reconnect (%ds remaining)...", r.graceRemaining)
				r.broadcastLocked("pause", msg)
			}
			r.mu.Unlock()
		}
	}
}

// Rejoin khôi phục kết nối cho người chơi rớt mạng quay lại phòng
func (r *Room) Rejoin(token string, newSendChan chan *pb.GameSync) (*RoomPlayer, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Status == RoomClosed {
		return nil, errors.New("match has already ended or closed")
	}

	player := r.getPlayerByToken(token)
	if player == nil {
		return nil, errors.New("unauthorized: player not in this match")
	}

	// Ngừng đếm ngược ân hạn nếu đang tạm dừng
	if r.Status == RoomPaused {
		if r.graceStopChan != nil {
			close(r.graceStopChan)
			r.graceStopChan = nil
		}
		r.Status = RoomActive
	}

	player.Connected = true
	player.SendChan = newSendChan

	resumeMsg := fmt.Sprintf("%s reconnected. Match resumed!", player.Username)
	// Gói resume này đã mang đầy đủ ma trận bàn cờ cập nhật cho cả 2 bên
	r.broadcastLocked("resume", resumeMsg)

	return player, nil
}

func (r *Room) updateEloOnGameEndLocked() {
	if r.eloUpdated || r.authMgr == nil {
		return
	}
	r.eloUpdated = true

	scoreWhite := 0.5
	if r.State.Winner == core.White {
		scoreWhite = 1.0
	} else if r.State.Winner == core.Black {
		scoreWhite = 0.0
	}

	newWhite, newBlack := core.CalculateElo(r.White.Elo, r.Black.Elo, scoreWhite)

	wonW, lostW, drewW := (scoreWhite == 1.0), (scoreWhite == 0.0), (scoreWhite == 0.5)
	wonB, lostB, drewB := (scoreWhite == 0.0), (scoreWhite == 1.0), (scoreWhite == 0.5)

	_ = r.authMgr.UpdateStats(r.White.Username, newWhite, wonW, lostW, drewW)
	_ = r.authMgr.UpdateStats(r.Black.Username, newBlack, wonB, lostB, drewB)

	r.White.Elo = newWhite
	r.Black.Elo = newBlack
}

func (r *Room) closeRoomLocked() {
	r.Status = RoomClosed
	if r.graceStopChan != nil {
		close(r.graceStopChan)
		r.graceStopChan = nil
	}
	if r.onRemove != nil {
		go r.onRemove(r.ID)
	}
}

func (r *Room) getPlayerByToken(token string) *RoomPlayer {
	if r.White.Token == token {
		return r.White
	}
	if r.Black.Token == token {
		return r.Black
	}
	return nil
}

func (r *Room) getOtherPlayer(token string) *RoomPlayer {
	if r.White.Token == token {
		return r.Black
	}
	if r.Black.Token == token {
		return r.White
	}
	return nil
}

func (r *Room) broadcastLocked(msgType, text string) {
	syncMsg := r.toGameSyncLocked(msgType, text)

	if r.White.Connected {
		select {
		case r.White.SendChan <- syncMsg:
		default:
		}
	}

	if r.Black.Connected {
		select {
		case r.Black.SendChan <- syncMsg:
		default:
		}
	}
}

func (r *Room) toGameSyncLocked(msgType, message string) *pb.GameSync {
	sync := &pb.GameSync{
		Type:                   msgType,
		CurrentTurn:            int32(r.State.CurrentTurn),
		Winner:                 int32(r.State.Winner),
		Message:                message,
		IsPaused:               r.Status == RoomPaused,
		GraceSecondsRemaining: int32(r.graceRemaining),
		WhiteElo:               int32(r.White.Elo),
		BlackElo:               int32(r.Black.Elo),
	}
	for _, row := range r.State.Board {
		sync.Board = append(sync.Board, &pb.BoardRow{Cells: row})
	}
	for _, pos := range r.State.WinningPositions {
		sync.WinningPositions = append(sync.WinningPositions, &pb.Position{X: int32(pos[0]), Y: int32(pos[1])})
	}
	return sync
}

// ==========================================
// ROOM MANAGER
// ==========================================

type RoomManager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Room),
	}
}

func (rm *RoomManager) Add(room *Room) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.rooms[room.ID] = room
}

func (rm *RoomManager) Get(id string) (*Room, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	room, exists := rm.rooms[id]
	return room, exists
}

func (rm *RoomManager) Remove(id string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.rooms, id)
}
