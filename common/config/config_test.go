package config_test

import (
	"config"
	"fmt"
	"testing"
)

func TestConfig(t *testing.T) {
	conf := config.Config{1, "2"}
	config.WriteConfig("testconfig.config", conf)

	conf2, err := config.ReadConfig("testconfig.config")
	if err != nil {
		panic(err)
	}
	fmt.Println(conf2.BatchSize)
	fmt.Println(conf2.Payload)
}
