package client

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type ServerConfig struct {
	ServerURL string            `json:"serverURL"`
	Type      string            `json:"type"`
	Headers   map[string]string `json:"headers,omitempty"`
}

var serverConfigs []ServerConfig

func LoadConfig() error {
	// 从 ConfigMap 中读取 server 配置
	data, err := ioutil.ReadFile("./config/config.json")
	if err != nil {
		fmt.Printf("Error reading config.json: %v\n", err)
		return err
	}

	err = json.Unmarshal(data, &serverConfigs)
	if err != nil {
		fmt.Printf("Error parsing config.json: %v\n", err)
		return err
	}
	return nil
}
