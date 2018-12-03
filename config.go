package main

import (
	"encoding/json"
	"errors"
	"os"
)

// Read JSON configuration from the file
func readConfigFile(filename string) (conf *Config, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decode json
	conf = &Config{}
	err = json.NewDecoder(file).Decode(conf)
	if err != nil {
		return nil, err
	}
	return conf, nil
}

// Check json params
func checkParams(conf *Config) error {
	if conf.Addr == "" {
		return errors.New("server's tcp address is required")
	}

	if conf.RestAddr == "" {
		return errors.New("server's rest http address is required")
	}

	if conf.LogFile == "" {
		return errors.New("server's log file is required")
	}
	return nil
}
