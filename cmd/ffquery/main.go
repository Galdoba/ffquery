package main

import (
	"context"
	"log"
	"os"

	"github.com/Galdoba/ffquery/internal/commands"
)

func main() {
	cmd := commands.Root()

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
