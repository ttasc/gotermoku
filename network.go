package main

import (
	"encoding/json"
	"io"
	"net"
)

// NetMessage represents a data packet exchanged between the Host and the Client via JSON.
type NetMessage struct {
	Type string `json:"type"` // Message type: "move", "sync", or "restart"

	// Payload for "move" messages
	X int `json:"x,omitempty"`
	Y int `json:"y,omitempty"`

	// Payload for "sync" messages (synchronizing state from the Server to the Client).
	// A pointer is used to avoid copying the entire board array when the message type is not "sync".
	Board            *[BoardRows][BoardCols]uint8 `json:"board,omitempty"`
	CurrentTurn      uint8                        `json:"turn,omitempty"`
	Winner           uint8                        `json:"winner,omitempty"`
	WinningPositions [][2]int                     `json:"winning_positions,omitempty"`
}

// NetworkManager manages the TCP connection and background goroutines for network I/O.
type NetworkManager struct {
	conn     net.Conn
	encoder  *json.Encoder
	Incoming chan NetMessage // Channel routing incoming messages from the background reading goroutine to the main thread
	IsHost   bool            // Indicates whether this instance is acting as the Server (Host) or the Client
}

// HostGame opens a TCP port, blocks until exactly ONE client connects, and then initializes the NetworkManager.
func HostGame(port string) (*NetworkManager, error) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	// Wait for the first and only client (blocks until a connection is established).
	conn, err := ln.Accept()
	if err != nil {
		ln.Close()
		return nil, err
	}

	// Since Gomoku is strictly a two-player game, close the listener immediately after receiving one connection.
	ln.Close()

	nm := &NetworkManager{
		conn:     conn,
		encoder:  json.NewEncoder(conn),
		Incoming: make(chan NetMessage, 10), // Small buffer to prevent TCP bottlenecks during rapid clicking
		IsHost:   true,
	}

	// Start a dedicated background goroutine for reading network streams.
	go nm.readLoop()

	return nm, nil
}

// JoinGame connects to an existing Host and initializes the NetworkManager.
func JoinGame(addr string) (*NetworkManager, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	nm := &NetworkManager{
		conn:     conn,
		encoder:  json.NewEncoder(conn),
		Incoming: make(chan NetMessage, 10),
		IsHost:   false,
	}

	// Start a dedicated background goroutine for reading network streams.
	go nm.readLoop()

	return nm, nil
}

// readLoop continuously reads and decodes JSON streams from the TCP connection in a dedicated background goroutine,
// pushing the results into the Incoming channel.
func (nm *NetworkManager) readLoop() {
	decoder := json.NewDecoder(nm.conn)
	for {
		var msg NetMessage
		err := decoder.Decode(&msg)
		if err != nil {
			if err == io.EOF {
				// Connection lost, notify the main thread and exit the loop.
				nm.Incoming <- NetMessage{Type: "disconnect"}
				break
			}
			// Ignore and skip corrupted JSON packets.
			continue
		}

		// Successfully decoded JSON, push it to the channel.
		nm.Incoming <- msg
	}
}

// Send encodes a NetMessage struct into JSON and transmits it over the TCP connection.
// It is safely called from the main thread.
func (nm *NetworkManager) Send(msg NetMessage) error {
	// json.Encoder provides internal buffering, making writes relatively safe.
	return nm.encoder.Encode(msg)
}

// Close releases network resources and terminates the TCP connection.
func (nm *NetworkManager) Close() {
	if nm.conn != nil {
		nm.conn.Close()
	}
	// Explicitly closing the channel here is omitted to prevent panics from the readLoop attempting to send
	// to a closed channel. Go's Garbage Collector will safely clean up the channel when the process terminates.
}
