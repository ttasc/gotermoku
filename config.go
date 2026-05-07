package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type GameConfig struct {
	Rows     int
	Cols     int
	IsOnline bool
	IsHost   bool
	JoinAddr string
	Port     string
}

// printUsage displays the command-line help message.
func printUsage() {
	fmt.Println("Gomoku TUI Game")
	fmt.Println("Usage:")
	fmt.Println("  gotermoku [options]")
	fmt.Println("\nOptions:")
	fmt.Println("  -h, --help       Show this help message")
	fmt.Println("  --host           Enable online mode and act as the Host (Server)")
	fmt.Println("  --join ADDRESS   Enable online mode and connect to a Host at ADDRESS")
	fmt.Println("  --port PORT      Specify the port to listen on or connect to (default: 3333)")
	fmt.Println("  -s, --size       Specify board size as ROWSxCOLS (default: 20x30, min: 3x3)")
	fmt.Println("\nExamples:")
	fmt.Println("  Offline Mode : gotermoku -s 15x15")
	fmt.Println("  Host a Game  : gotermoku --host --port 9000")
	fmt.Println("  Join a Game  : gotermoku --join 192.168.1.10 --port 9000")
}

func ParseConfig() *GameConfig {
	var (
		helpFlag bool
		hFlag    bool
		hostFlag bool
		joinAddr string
		port     string
		sizeStr  string
	)

	flag.BoolVar(&helpFlag, "help", false, "Show help message")
	flag.BoolVar(&hFlag, "h", false, "Show help message")
	flag.BoolVar(&hostFlag, "host", false, "Act as Host")
	flag.StringVar(&joinAddr, "join", "", "IP address to join")
	flag.StringVar(&port, "port", "3333", "Port to use")
	flag.StringVar(&sizeStr, "size", "20x30", "Board size (ROWSxCOLS)")
	flag.StringVar(&sizeStr, "s", "20x30", "Board size (shorthand)")

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

	parts := strings.Split(strings.ToLower(sizeStr), "x")
	if len(parts) != 2 {
		fmt.Println("Error: Invalid size format. Use ROWSxCOLS (e.g., 20x30).")
		os.Exit(1)
	}

	rows, errR := strconv.Atoi(parts[0])
	cols, errC := strconv.Atoi(parts[1])
	if errR != nil || errC != nil || rows < 3 || cols < 3 {
		fmt.Println("Error: Invalid dimensions. Minimum size is 3x3.")
		os.Exit(1)
	}

	return &GameConfig{
		Rows:     rows,
		Cols:     cols,
		IsOnline: hostFlag || joinAddr != "",
		IsHost:   hostFlag,
		JoinAddr: joinAddr,
		Port:     port,
	}
}
