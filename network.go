package main

import (
	"encoding/json"
	"io"
	"net"
)

type NetMessage struct {
	Type             string     `json:"type"`
	X                int        `json:"x,omitempty"`
	Y                int        `json:"y,omitempty"`
	Board            [][]uint8  `json:"board,omitempty"`
	CurrentTurn      uint8      `json:"turn,omitempty"`
	Winner           uint8      `json:"winner,omitempty"`
	WinningPositions [][2]int   `json:"winning_positions,omitempty"`
}

type NetworkManager struct {
	conn     net.Conn
	encoder  *json.Encoder
	Incoming chan NetMessage
	IsHost   bool
}

func HostGame(port string) (*NetworkManager, error) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	conn, err := ln.Accept()
	if err != nil {
		ln.Close()
		return nil, err
	}
	ln.Close()

	return startNetworkManager(conn, true), nil
}

func JoinGame(addr string) (*NetworkManager, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return startNetworkManager(conn, false), nil
}

func startNetworkManager(conn net.Conn, isHost bool) *NetworkManager {
	nm := &NetworkManager{
		conn:     conn,
		encoder:  json.NewEncoder(conn),
		Incoming: make(chan NetMessage, 10),
		IsHost:   isHost,
	}
	go nm.readLoop()
	return nm
}

func (nm *NetworkManager) readLoop() {
	decoder := json.NewDecoder(nm.conn)
	for {
		var msg NetMessage
		if err := decoder.Decode(&msg); err != nil {
			if err == io.EOF {
				nm.Incoming <- NetMessage{Type: "disconnect"}
				break
			}
			continue
		}
		nm.Incoming <- msg
	}
}

func (nm *NetworkManager) Send(msg NetMessage) error {
	return nm.encoder.Encode(msg)
}

func (nm *NetworkManager) Close() {
	if nm.conn != nil {
		nm.conn.Close()
	}
}
