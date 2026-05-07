package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ttasc/ttbox"
)

// Board dimensions and rendering constants.
const (
	BoardRows  = 20
	BoardCols  = 30
	BoardWidth = BoardCols * 3
	BoardHeigh = BoardRows
)

// printUsage displays the command-line help message and usage instructions.
func printUsage() {
	fmt.Println("Gomoku TUI Game")
	fmt.Println("Usage:")
	fmt.Println("  gotermoku [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  -h, --help       Show this help message")
	fmt.Println("  --host           Enable online mode and act as the Host (Server)")
	fmt.Println("  --join ADDRESS   Enable online mode and connect to a Host at ADDRESS")
	fmt.Println("  --port PORT      Specify the port to listen on or connect to (default: 3333)")
	fmt.Println("\nExamples:")
	fmt.Println("  Offline Mode : gotermoku")
	fmt.Println("  Host a Game  : gotermoku --host --port 9000")
	fmt.Println("  Join a Game  : gotermoku --join 192.168.1.10 --port 9000")
}

func main() {

	var (
		helpFlag bool
		hFlag    bool
		hostFlag bool
		joinAddr string
		port     string
	)

	flag.BoolVar(&helpFlag, "help", false, "Show help message")
	flag.BoolVar(&hFlag, "h", false, "Show help message")
	flag.BoolVar(&hostFlag, "host", false, "Act as Host")
	flag.StringVar(&joinAddr, "join", "", "IP address to join")
	flag.StringVar(&port, "port", "3333", "Port to use")

	flag.Usage = printUsage
	flag.Parse()

	if helpFlag || hFlag {
		printUsage()
		os.Exit(0)
	}

	if hostFlag && joinAddr != "" {
		fmt.Println("Error: Cannot use both --host and --join at the same time.")
		printUsage()
		os.Exit(1)
	}

	isOnline := hostFlag || joinAddr != ""
	isHost := hostFlag

	var netMgr *NetworkManager
	playerColor := uint8(White) // Default mode (Offline or Host) always plays as White.

	if isOnline {
		var err error
		if isHost {
			fmt.Printf("Starting Host... Waiting for client to connect on port %s...\n", port)
			netMgr, err = HostGame(port)
		} else {
			address := fmt.Sprintf("%s:%s", joinAddr, port)
			fmt.Printf("Connecting to Host at %s...\n", address)
			netMgr, err = JoinGame(address)
			playerColor = Black // Client always plays as Black.
		}

		if err != nil {
			fmt.Printf("Network error: %v\n", err)
			os.Exit(1)
		}
		defer netMgr.Close()
	}

	var disconnectMsg string

	defer func() {
		if disconnectMsg != "" {
			fmt.Printf("\n\nNOTICE: %s\n\n", disconnectMsg)
		}
	}()

	state := NewGameState()
	// Pass context info into the state
	state.IsOnline = isOnline
	state.LocalPlayerColor = playerColor

	if err := ttbox.Init(); err != nil {
		fmt.Printf("Error initializing TUI: %v\n", err)
		os.Exit(1)
	}
	defer ttbox.Close()
	ttbox.EnableMouse()
	defer ttbox.DisableMouse()

	isRunning := true
	for isRunning {

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
				// No incoming messages, continue running to prevent freezing the game loop.
			}
		}

		evt, err := ttbox.PollEventTimeout(16 * time.Millisecond)
		if err == nil {
			switch evt.Type {
			case ttbox.EventKey:
				// Handle application exit.
				if evt.Key == ttbox.KeyEscape || evt.Key == ttbox.KeyCtrlC || evt.Ch == 'q' || evt.Ch == 'Q' {
					isRunning = false
				}
				// Handle game restart.
				if state.Winner != Empty && (evt.Ch == 'r' || evt.Ch == 'R') {
					if netMgr != nil {
						if netMgr.IsHost {
							state.Reset()
							broadcastSync(state, netMgr) // Host initiates the reset and syncs with the client.
						} else {
							netMgr.Send(NetMessage{Type: "restart"}) // Client requests a restart from the Host.
						}
					} else {
						state.Reset() // Offline mode resets immediately.
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

// handleNetworkMessage processes incoming network messages and updates the game state accordingly.
func handleNetworkMessage(msg NetMessage, state *GameState, netMgr *NetworkManager) {
	switch msg.Type {
	case "move":
		if netMgr.IsHost {
			// [SERVER] The client requests to place a piece.
			// Safety validation: Ensure it is the Client's turn (Black), the cell is empty, and the game is not over.
			if state.CurrentTurn == Black && state.Board[msg.Y][msg.X] == Empty && state.Winner == Empty {
				state.Board[msg.Y][msg.X] = Black
				state.MoveCount[Black]++
				checkWin(state, msg.X, msg.Y)
				if state.Winner == Empty {
					state.CurrentTurn = White
				}
				// After updating, broadcast the new state to the client to ensure synchronization.
				broadcastSync(state, netMgr)
			}
		}

	case "sync":
		if !netMgr.IsHost {
			// [CLIENT] Completely overwrite local state with data from the Server (the Source of Truth).
			state.Board = *msg.Board // Dereference the pointer to copy the board values.
			state.CurrentTurn = msg.CurrentTurn
			state.Winner = msg.Winner
			state.WinningPositions = msg.WinningPositions
		}

	case "restart":
		if netMgr.IsHost {
			// Client requests a restart; the Server resets the state and forces the Client to sync.
			state.Reset()
			broadcastSync(state, netMgr)
		}
	}
}

// broadcastSync sends the current game state from the Host to the Client to ensure both sides are synchronized.
func broadcastSync(state *GameState, netMgr *NetworkManager) {
	netMgr.Send(NetMessage{
		Type:             "sync",
		Board:            &state.Board,
		CurrentTurn:      state.CurrentTurn,
		Winner:           state.Winner,
		WinningPositions: state.WinningPositions,
	})
}
