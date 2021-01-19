package server

import (
	"os"
	"fmt"
	"encoding/json"
	"io/ioutil"
	"errors"
)

type AddressConfig struct {
	Http	string `json:"http"`
	Tcp	string `json:"tcp"`
}


type Config struct {
	Addresses AddressConfig `json:"addresses"`
	DataDir	  string	`json:"data_dir"`
	Name	  string	`json:"name"`
}

func defaultName() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", errors.New("No name provided and error reading hostname")
	}
	return "taylor.server." + hostname, nil
}

func DevModeConfig() Config {
	name, _ := defaultName()
	config := Config{
		Addresses: AddressConfig{
			Http: "127.0.0.1:8400",
			Tcp:  "127.0.0.1:8401",
		},
		DataDir: ".taylor-dev-temp/",
		Name: name,
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

	if config.Addresses.Http == "" {
		config.Addresses.Http = "127.0.0.1:8400"
	}
	if config.Addresses.Tcp == "" {
		config.Addresses.Tcp = "127.0.0.1:8401"
	}
	if config.DataDir == "" {
		return config, errors.New("No data_dir specified")
	}
	if config.Name == "" {
		config.Name, err = defaultName()
		if err != nil {
			return config, err
		}
	}

	fmt.Printf("%+v\n", config)

	return config, nil
}
