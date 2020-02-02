package server

import (
	"os"
	"fmt"

	"taylor/server/database"
	"taylor/lib/structs"
)

func Run() int {
	os.RemoveAll("/tmp/taylor-db-1.db")
	store, err := database.Open("/tmp/taylor-db-1.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Db Error: %v\n", err)
		return 1
	}
	query := fmt.Sprintf(`
	SELECT * FROM jobs ORDER BY ts ASC
	`)

	jobs, err := store.CollectQuery(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query Error: %v\n", err)
		return 1
	}

	fmt.Printf("All: %v\n", jobs)

	jobs, err = store.JobsWithStatus(structs.JOB_STATUS_SCHEDULED, 10)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query Error: %v\n", err)
		return 1
	}

	fmt.Printf("SCHEDULED: %v\n", jobs)

	StartScheduler()

	config, err := ReadConfig("./config.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Read Config Error: %v\n", err)
		return 1
	}

	tcpS, err := StartTcp(*config, TcpDependencies{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Starting Tcp: %v\n", err)
		return 1
	}

	deps := ApiDependencies{
		Store: store,
		TcpServer: tcpS,
	}

	err = StartApi(config, &deps)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error Starting Api: %v\n", err)
		return 1
	}

	return 0
}
