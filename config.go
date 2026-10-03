package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
)

type GameConfig struct {
	Rows      int
	Cols      int
	IsBotMode bool
	IsOnline  bool
	IsHost    bool
	JoinAddr  string
	Port      string
}

func parseDimensions(sizeStr string) (int, int, error) {
	parts := strings.Split(strings.ToLower(sizeStr), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid size format. Use ROWSxCOLS (e.g., 20x30)")
	}
	rows, errR := strconv.Atoi(parts[0])
	cols, errC := strconv.Atoi(parts[1])
	if errR != nil || errC != nil || rows < 3 || cols < 3 {
		return 0, 0, fmt.Errorf("invalid dimensions. Minimum size is 3x3")
	}
	return rows, cols, nil
}

func ParseConfig(args []string) (*GameConfig, error) {
	fs := flag.NewFlagSet("gotermoku", flag.ContinueOnError)
	fs.Usage = func() {} // Suppress default flag output

	helpFlag := fs.Bool("help", false, "Show help message")
	hFlag := fs.Bool("h", false, "Show help message")
	hostFlag := fs.Bool("host", false, "Act as Host")
	botFlag := fs.Bool("bot", false, "Play against AI")
	joinAddr := fs.String("join", "", "IP address to join")
	port := fs.String("port", "3333", "Port to use")
	sizeStr := fs.String("size", "20x30", "Board size (ROWSxCOLS)")
	sizeShorthand := fs.String("s", "20x30", "Board size (shorthand)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if *helpFlag || *hFlag {
		return nil, fmt.Errorf("help") // Signal main to print help
	}
	if *botFlag && (*hostFlag || *joinAddr != "") {
		return nil, fmt.Errorf("cannot use --bot with online modes (--host or --join)")
	}
	if *hostFlag && *joinAddr != "" {
		return nil, fmt.Errorf("cannot use both --host and --join at the same time")
	}

	size := *sizeStr
	if *sizeShorthand != "20x30" {
		size = *sizeShorthand
	}

	rows, cols, err := parseDimensions(size)
	if err != nil {
		return nil, err
	}

	return &GameConfig{
		Rows:      rows,
		Cols:      cols,
		IsOnline:  *hostFlag || *joinAddr != "",
		IsHost:    *hostFlag,
		IsBotMode: *botFlag,
		JoinAddr:  *joinAddr,
		Port:      *port,
	}, nil
}

func PrintUsage() {
	fmt.Println("Gomoku TUI Game\nUsage:\n  gotermoku [options]\nOptions:")
	fmt.Println("  -h, --help       Show this help message")
	fmt.Println("  -s, --size       Specify board size as ROWSxCOLS (default: 20x30, min: 3x3)")
	fmt.Println("  --bot            Play against AI in offline mode")
	fmt.Println("  --host           Enable online mode and act as the Host (Server)")
	fmt.Println("  --join ADDRESS   Enable online mode and connect to a Host at ADDRESS")
	fmt.Println("  --port PORT      Specify the port to listen on or connect to (default: 3333)")
}
