package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ttasc/ttbox"
)

func RunGame(state *GameState, netMgr *NetworkManager) {
	if err := ttbox.Init(); err != nil {
		fmt.Printf("Error initializing TUI: %v\n", err)
		os.Exit(1)
	}
	defer ttbox.Close()

	// Validate terminal size
	termW, termH := ttbox.Size()
	maxCols := termW / CellWidth
	maxRows := termH - 6

	if state.Cols > maxCols || state.Rows > maxRows {
		ttbox.Close()
		fmt.Printf("Error: Size %dx%d is too large for your terminal.\n", state.Rows, state.Cols)
		fmt.Printf("Your terminal's max capacity is %dx%d.\n", maxRows, maxCols)
		fmt.Printf("Please maximize your terminal window, zoom out, or choose a smaller size.\n")
		os.Exit(1)
	}

	ttbox.EnableMouse()
	defer ttbox.DisableMouse()

	if state.IsOnline && netMgr != nil && netMgr.IsHost {
		broadcastSync(state, netMgr)
	}

	var disconnectMsg string
	defer func() {
		if disconnectMsg != "" {
			fmt.Printf("\n\nNOTICE: %s\n\n", disconnectMsg)
		}
	}()

	playerColor := state.LocalPlayerColor
	isRunning := true

	// Channel giao tiếp với Bot và cờ chống trùng lặp
	botMoves := make(chan [2]int, 1)
	botThinking := false

	for isRunning {
		// --- BỔ SUNG LOGIC GỌI BOT TẠI ĐÂY ---
		if state.IsBotMode && state.CurrentTurn == Black && state.Winner == Empty && !botThinking {
			botThinking = true
			go func(currentState *GameState) {
				// Giả lập thời gian bot "suy nghĩ" để UX mượt mà (0.4s)
				time.Sleep(400 * time.Millisecond)
				move := getBotMove(currentState)
				botMoves <- move
			}(state)
		}

		// Nhận nước đi của Bot (Không block UI)
		select {
		case move := <-botMoves:
			botThinking = false
			tryPlacePiece(state, netMgr, Black, move[0], move[1]) // Bot cầm Đen
		default:
		}
		// Network Handler
		if netMgr != nil {
			select {
			case msg := <-netMgr.Incoming:
				if msg.Type == "disconnect" {
					isRunning = false
					disconnectMsg = "The opponent has disconnected."
					continue
				}
				handleNetworkMessage(msg, state, netMgr)
			default:
				// none-blocking game
			}
		}

		// Input Handler
		evt, err := ttbox.PollEventTimeout(500 * time.Millisecond)
		if err == nil {
			switch evt.Type {
			case ttbox.EventKey:
				if evt.Key == ttbox.KeyEscape || evt.Key == ttbox.KeyCtrlC || evt.Ch == 'q' || evt.Ch == 'Q' {
					isRunning = false
				}
				if state.Winner != Empty && (evt.Ch == 'r' || evt.Ch == 'R') {
					if netMgr != nil {
						if netMgr.IsHost {
							state.Reset()
							broadcastSync(state, netMgr)
						} else {
							netMgr.Send(NetMessage{Type: "restart"})
						}
					} else {
						state.Reset()
					}
				} else {
					handleKeyboard(evt, state, netMgr, playerColor)
				}
			case ttbox.EventMouse:
				handleMouse(evt, state, netMgr, playerColor)
			}
		}

		Render(state)
	}
}
