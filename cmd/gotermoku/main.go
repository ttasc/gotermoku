package main

import (
	"os"

	"github.com/ttasc/gotermoku/internal/client"
)

func main() {
	client.Run(os.Args[1:])
}
