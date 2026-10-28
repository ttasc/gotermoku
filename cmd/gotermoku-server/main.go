package main

import (
	"os"

	"github.com/ttasc/gotermoku/internal/server"
)

func main() {
	server.Run(os.Args[1:])
}
