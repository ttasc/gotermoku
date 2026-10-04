package netio

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"

	"github.com/ttasc/gotermoku/internal/config"
	"github.com/ttasc/gotermoku/internal/core"
)

type Transport interface {
	Send(msg NetMessage) error
	Receive() <-chan NetMessage
	Close()
}

type NetMessage struct {
	Type             string     `json:"type"`
	X                int        `json:"x,omitempty"`
	Y                int        `json:"y,omitempty"`
	Board            [][]uint8  `json:"board,omitempty"`
	CurrentTurn      uint8      `json:"turn,omitempty"`
	Winner           uint8      `json:"winner,omitempty"`
	WinningPositions [][2]int   `json:"winning_positions,omitempty"`
}

type TCPTransport struct {
	conn     net.Conn
	encoder  *json.Encoder
	incoming chan NetMessage
}

func InitTransport(cfg *config.Config) Transport {
	var conn net.Conn
	var err error

	if cfg.IsHost {
		fmt.Printf("Starting Host... Waiting for client to connect on port %s...\n", cfg.Port)
		ln, lnErr := net.Listen("tcp", ":"+cfg.Port)
		if lnErr != nil {
			fmt.Printf("Network error: %v\n", lnErr)
			os.Exit(1)
		}
		conn, err = ln.Accept()
		ln.Close()
	} else {
		addr := fmt.Sprintf("%s:%s", cfg.JoinAddr, cfg.Port)
		fmt.Printf("Connecting to Host at %s...\n", addr)
		conn, err = net.Dial("tcp", addr)
	}

	if err != nil {
		fmt.Printf("Connection error: %v\n", err)
		os.Exit(1)
	}

	t := &TCPTransport{
		conn:     conn,
		encoder:  json.NewEncoder(conn),
		incoming: make(chan NetMessage, 10),
	}
	go t.readLoop()
	return t
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

func (t *TCPTransport) readLoop() {
	decoder := json.NewDecoder(t.conn)
	for {
		var msg NetMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				t.incoming <- NetMessage{Type: "disconnect"}
				break
			}
			continue
		}
		t.incoming <- msg
	}
}

func (t *TCPTransport) Send(msg NetMessage) error {
	return t.encoder.Encode(msg)
}

func (t *TCPTransport) Receive() <-chan NetMessage {
	return t.incoming
}

func (t *TCPTransport) Close() {
	if t.conn != nil {
		t.conn.Close()
	}
}
