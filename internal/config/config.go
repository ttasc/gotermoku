package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Rows          int
	Cols          int
	IsBotMode     bool
	IsOnline      bool
	IsHost        bool
	JoinAddr      string
	Port          string
	// Các trường phục vụ Central Server Online
	ServerAddr    string
	Username      string
	Password      string
	IsRegister    bool
	IsCentralMode bool
	AssignedColor uint8
	OpponentName  string
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

func Parse(args []string) *Config {
	fs := flag.NewFlagSet("gotermoku", flag.ContinueOnError)
	fs.Usage = func() {}

	helpFlag := fs.Bool("help", false, "")
	hFlag := fs.Bool("h", false, "")
	hostFlag := fs.Bool("host", false, "")
	botFlag := fs.Bool("bot", false, "")
	joinAddr := fs.String("join", "", "")
	port := fs.String("port", "3333", "")
	sizeStr := fs.String("size", "20x30", "")
	sizeShorthand := fs.String("s", "20x30", "")

	// Cờ mới cho Central Server
	serverAddr := fs.String("server", "", "")
	username := fs.String("user", "", "")
	password := fs.String("pass", "", "")
	registerFlag := fs.Bool("register", false, "")

	if err := fs.Parse(args); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if *helpFlag || *hFlag {
		printUsage()
		os.Exit(0)
	}

	// Kiểm tra loại trừ xung đột các chế độ chơi
	isCentral := *serverAddr != ""
	if isCentral && (*botFlag || *hostFlag || *joinAddr != "") {
		fmt.Println("Error: cannot combine --server with --bot, --host, or --join")
		os.Exit(1)
	}
	if *botFlag && (*hostFlag || *joinAddr != "") {
		fmt.Println("Error: cannot use --bot with online modes")
		os.Exit(1)
	}
	if *hostFlag && *joinAddr != "" {
		fmt.Println("Error: cannot use both --host and --join")
		os.Exit(1)
	}

	size := *sizeStr
	if *sizeShorthand != "20x30" {
		size = *sizeShorthand
	}

	rows, cols, err := parseDimensions(size)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	return &Config{
		Rows:          rows,
		Cols:          cols,
		IsOnline:      *hostFlag || *joinAddr != "" || isCentral,
		IsHost:        *hostFlag,
		IsBotMode:     *botFlag,
		JoinAddr:      *joinAddr,
		Port:          *port,
		ServerAddr:    *serverAddr,
		Username:      *username,
		Password:      *password,
		IsRegister:    *registerFlag,
		IsCentralMode: isCentral,
	}
}

func printUsage() {
	fmt.Println("Gomoku TUI Game\nUsage:\n  gotermoku [options]\nOptions:")
	fmt.Println("  -h, --help          Show this help message")
	fmt.Println("  -s, --size          Specify board size as ROWSxCOLS (default: 20x30, min: 3x3)")
	fmt.Println("  --bot               Play against AI in offline mode")
	fmt.Println("  --host              Enable online mode and act as Host (LAN P2P)")
	fmt.Println("  --join ADDRESS      Enable online mode and connect to Host (LAN P2P)")
	fmt.Println("  --port PORT         Specify port for LAN P2P (default: 3333)")
	fmt.Println("\nCentral Server Online Options:")
	fmt.Println("  --server HOST:PORT  Connect to Central Server for online matchmaking")
	fmt.Println("  --user USERNAME     Account username")
	fmt.Println("  --pass PASSWORD     Account password")
	fmt.Println("  --register          Register account instead of login")
}
