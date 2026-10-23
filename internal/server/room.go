package server

import (
	"errors"
	"sync"

	"github.com/ttasc/gotermoku/internal/core"
	"github.com/ttasc/gotermoku/internal/netio/pb"
)

type RoomPlayer struct {
	Username string
	Token    string
	Color    uint8
	SendChan chan *pb.GameSync
}

type Room struct {
	ID       string
	mu       sync.Mutex
	State    *core.GameState
	White    *RoomPlayer
	Black    *RoomPlayer
	Closed   bool
	onRemove func(roomID string)
}

func NewRoom(id string, whiteUser, whiteToken, blackUser, blackToken string, rows, cols int, onRemove func(string)) *Room {
	r := &Room{
		ID:    id,
		State: core.NewGameState(rows, cols),
		White: &RoomPlayer{
			Username: whiteUser,
			Token:    whiteToken,
			Color:    core.White,
			SendChan: make(chan *pb.GameSync, 10),
		},
		Black: &RoomPlayer{
			Username: blackUser,
			Token:    blackToken,
			Color:    core.Black,
			SendChan: make(chan *pb.GameSync, 10),
		},
		onRemove: onRemove,
	}
	return r
}

func (r *Room) BroadcastInitialState() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.broadcastLocked("sync", "")
}

func (r *Room) HandleAction(token string, action *pb.ClientAction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Closed {
		return errors.New("room is closed")
	}

	player := r.getPlayerByToken(token)
	if player == nil {
		return errors.New("unauthorized action: player not in this room")
	}

	switch action.Type {
	case "move":
		// Chỉ cho phép đi khi đúng lượt
		if r.State.CurrentTurn != player.Color {
			return errors.New("not your turn")
		}
		x, y := int(action.X), int(action.Y)
		if !core.CanPlacePiece(r.State, x, y) {
			return errors.New("invalid move coordinate")
		}
		core.PlacePiece(r.State, x, y, player.Color)
		r.broadcastLocked("sync", "")

	case "restart":
		// Chỉ cho phép restart khi ván đấu đã kết thúc
		if r.State.Winner != core.Empty {
			r.State.Reset()
			r.broadcastLocked("sync", "")
		}
	}

	return nil
}

func (r *Room) HandleDisconnect(token string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Closed {
		return
	}
	r.Closed = true

	leaver := r.getPlayerByToken(token)
	msg := "Opponent disconnected."
	if leaver != nil {
		msg = leaver.Username + " disconnected."
	}

	r.broadcastLocked("disconnect", msg)
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

func (r *Room) broadcastLocked(msgType, text string) {
	syncMsg := r.toGameSync(msgType, text)

	// Gửi non-blocking tới cả 2 người chơi để tránh nghẽn
	select {
	case r.White.SendChan <- syncMsg:
	default:
	}

	select {
	case r.Black.SendChan <- syncMsg:
	default:
	}
}

func (r *Room) toGameSync(msgType, message string) *pb.GameSync {
	sync := &pb.GameSync{
		Type:        msgType,
		CurrentTurn: int32(r.State.CurrentTurn),
		Winner:      int32(r.State.Winner),
		Message:     message,
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
