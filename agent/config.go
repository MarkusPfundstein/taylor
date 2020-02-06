package agent 

import (
	"os"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"errors"
)

type SchedulerConfig struct {
	MaxParallelJobs uint		`json:"max_parallel_jobs"`
}

type Config struct {
	ClusterAddr	string		`json:"cluster"`
	Name		string		`json:"name"`
	Capabilities	[]string	`json:"capabilities"`
	Scheduler	SchedulerConfig	`json:"scheduler"`
}

func ReadConfig(path string) (Config, error) {
	var config Config

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	if config.ClusterAddr == "" {
		return config, errors.New("no cluster address found in config")
	}

	if config.Name == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return config, errors.New("No name provided and error reading hostname")
		}
		config.Name = "taylor.agent." + hostname
	}

	if config.Scheduler.MaxParallelJobs == 0 {
		config.Scheduler.MaxParallelJobs = 1
	}
	if config.Capabilities == nil {
		config.Capabilities = make([]string, 0)
	}

	fmt.Printf("%+v\n", config)

	return config, nil
}
