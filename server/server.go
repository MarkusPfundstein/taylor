package server

import (
	"os"
	"fmt"
	"path"

	"taylor/server/database"
)

func Run(args []string) int {

	config, err := ReadConfig("./server-config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Read Config Error: %v\n", err)
		return 1
	}
	if _, err := os.Stat(config.DataDir); os.IsNotExist(err) {
		// create DataDir
		err = os.MkdirAll(config.DataDir, os.ModePerm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating data_dir: %s\n", config.DataDir)
			return 1
		}
	}

	loggingDir := path.Join(config.DataDir, "/job_logs")
	if _, err := os.Stat(loggingDir); os.IsNotExist(err) {
		err = os.MkdirAll(loggingDir, os.ModePerm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating logging directory: %s\n", loggingDir)
			return 1
		}
	}

	dataBaseDir := path.Join(config.DataDir, "taylor.db")
	// for now in dev mode
	os.RemoveAll(dataBaseDir)
	store, err := database.Open(dataBaseDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Db Error: %v\n", err)
		return 1
	}

	diskLog := NewDiskLog(loggingDir)

	tcpS, err := StartTcp(config, TcpDependencies{Store: store, DiskLog: diskLog})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Starting Tcp: %v\n", err)
		return 1
	}

	StartScheduler(config, store, tcpS)

	deps := ApiDependencies{
		Store:		store,
		TcpServer:	tcpS,
		DiskLog:	diskLog,
	}

	// from here on, we will block forever
	err = StartApi(config, deps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Starting Api: %v\n", err)
		return 1
	}

	return 0
}
