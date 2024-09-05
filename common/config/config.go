package config

import (
	"encoding/json"
	"io"
	"os"
)

// BatchSize is default size
const BatchSize = 128

// Config: the config of system
type Config struct {
	BatchSize int    `json:"batchSize"`
	Payload   string `json:"payload"`
}

// ReadConfig: read config file
func ReadConfig(filename string) (Config, error) {
	var config Config

	// open config file
	file, err := os.Open(filename)
	if err != nil {
		return config, err
	}
	defer file.Close()

	// read the config file
	data, err := io.ReadAll(file)
	if err != nil {
		return config, err
	}

	// decode JSON data
	err = json.Unmarshal(data, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}

// WriteConfig: write config file
func WriteConfig(filename string, config Config) error {

	// encode the data to json
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return err
	}

	// 写入配置文件
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
