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
	Name	  string	`json:"name"`
}

func ReadConfig(path string) (*Config, error) {
	var config Config

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	if config.Addresses.Http == "" {
		return nil, errors.New("No addresses.http found in config")
	}
	if config.Addresses.Tcp == "" {
		return nil, errors.New("No addresses.tcp found in config")
	}
	if config.Name == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, errors.New("No name provided and error reading hostname")
		}
		config.Name = "taylor.server." + hostname
	}

	fmt.Printf("%+v\n", config)

	return &config, nil
}
