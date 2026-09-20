package main

import (
	"fmt"
	"os"
	"qv/commands"
)

func main() {
	if err := commands.Run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "qv:", err)
		os.Exit(1)
	}
}
