package server

import (
	"os"
	"fmt"

	"taylor/server/database"
)

func Run(args []string) int {

	config, err := ReadConfig("./server-config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Read Config Error: %v\n", err)
		return 1
	}

	// for now in dev mode
	os.RemoveAll("/tmp/taylor-db-1.db")
	store, err := database.Open("/tmp/taylor-db-1.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Db Error: %v\n", err)
		return 1
	}

	tcpS, err := StartTcp(*config, TcpDependencies{Store: store})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Starting Tcp: %v\n", err)
		return 1
	}

	StartScheduler(*config, store, tcpS)

	deps := ApiDependencies{
		Store: store,
		TcpServer: tcpS,
	}

	// from here on, we will block forever
	err = StartApi(config, &deps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Starting Api: %v\n", err)
		return 1
	}

	return 0
}
