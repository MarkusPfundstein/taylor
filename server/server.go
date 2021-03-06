package server

import (
	"os"
	"fmt"
	"path"

	"taylor/server/database"
)

func Run(args []string, devMode bool) int {

	var config Config
	if devMode == true {
		config = DevModeConfig()
		os.RemoveAll(config.DataDir)
	} else {
		var configPath string
		if len(args) > 0 {
			configPath = args[0]
		} else {
			configPath = "./server-config.json"
		}

		configTmp, err := ReadConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Read Config Error: %v\n", err)
			return 1
		}
		config = configTmp
	}

	if _, err := os.Stat(config.DataDir); os.IsNotExist(err) {
		fmt.Printf("create data_dir: %s\n", config.DataDir)
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
