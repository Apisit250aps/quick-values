package main

import (
	"os"
	"qv/commands"
)

func main() {
	if err := commands.Run(os.Args[1:], os.Stdout); err != nil {
		commands.PrintError(os.Stderr, err)
		os.Exit(1)
	}
}
