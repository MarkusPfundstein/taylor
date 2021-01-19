package agent

import (
	"os"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"errors"
	"time"
)

type SchedulerConfig struct {
	MaxParallelJobs uint		`json:"max_parallel_jobs"`
}

type NvidiaConfig struct {
	NvidiaSmiPath string		`json:"nvidia_smi"`
	PollTimeMs    time.Duration	`json:"poll_ms"`
}

type Config struct {
	ClusterAddr	string		`json:"cluster"`
	Name		string		`json:"name"`
	Capabilities	[]string	`json:"capabilities"`
	Scheduler	SchedulerConfig	`json:"scheduler"`
	NvidiaCfg	NvidiaConfig	`json:"nvidia"`
}

func defaultName() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", errors.New("No name provided and error reading hostname")
	}
	return "taylor.agent." + hostname, nil
}

func DevModeConfig() Config {
	name, _ := defaultName()
	config := Config{
		ClusterAddr: "127.0.0.1:8401",
		Name: name,
		Capabilities: []string{},
		Scheduler: SchedulerConfig{
			MaxParallelJobs: 25,
		},
		NvidiaCfg: NvidiaConfig {
			NvidiaSmiPath: "nvidia-smi.exe",
			PollTimeMs: 1000,
		},
	}
	return config
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
		config.Name, err = defaultName()
		if err != nil {
			return config, err
		}
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
