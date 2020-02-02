package main

import (
	"os"
	"fmt"

	"taylor/server"
	"taylor/agent"
)

func _main() int {
	args := os.Args[1:]

	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "No arguments\n")
		return 1
	}

	if args[0] == "server" {
		return server.Run()
	} else if args[0] == "agent" {
		return agent.Run()
	} else {
		fmt.Println("Server or Agent?")
		return 1
	}
}

func main() {
	os.Exit(_main())
}
