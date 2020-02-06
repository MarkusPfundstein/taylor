package main

import (
	"os"
	"fmt"
	"time"

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
		return server.Run(args[1:], false)
	} else if args[0] == "agent" {
		return agent.Run(args[1:], false)
	} else if args[0] == "-dev" {
		go func() {
			// wait a bit for server to start up
			time.Sleep(1 * time.Second)
			agent.Run(args[1:], true)
		}()
		errServer := server.Run(args[1:], true)
		return errServer
	} else {
		fmt.Println("Server or Agent?")
		return 1
	}
}

func main() {
	os.Exit(_main())
}
