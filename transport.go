package main

import (
	"encoding/json"
	"io"
	"net"
)

// GameTransport decouples the network protocol from the game loop.
type GameTransport interface {
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

// TCPTransport is a concrete implementation of GameTransport.
type TCPTransport struct {
	conn     net.Conn
	encoder  *json.Encoder
	incoming chan NetMessage
}

func NewTCPTransport(conn net.Conn) *TCPTransport {
	t := &TCPTransport{
		conn:     conn,
		encoder:  json.NewEncoder(conn),
		incoming: make(chan NetMessage, 10),
	}
	go t.readLoop()
	return t
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
