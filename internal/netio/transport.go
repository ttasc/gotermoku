// internal/netio/transport.go
package netio

import (
	"github.com/ttasc/gotermoku/internal/core"
)

// Transport giữ nguyên thiết kế chuẩn để ẩn đi chi tiết của tầng Network
type Transport interface {
	Send(msg NetMessage) error
	Receive() <-chan NetMessage
	Close()
}

type NetMessage struct {
	Type             string     // "sync", "move", "restart", "disconnect"
	X                int
	Y                int
	Board            [][]uint8
	CurrentTurn      uint8
	Winner           uint8
	WinningPositions [][2]int
}

func BroadcastSync(state *core.GameState, t Transport) {
	if t == nil {
		return
	}
	t.Send(NetMessage{
		Type:             "sync",
		Board:            state.Board,
		CurrentTurn:      state.CurrentTurn,
		Winner:           state.Winner,
		WinningPositions: state.WinningPositions,
	})
}
