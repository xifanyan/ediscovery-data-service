package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	ADP struct {
		Domain   string `json:"domain"`
		User     string `json:"user"`
		Password string `json:"password"`
		Port     int    `json:"port"`
	} `json:"adp"`
	SearchWebAPI struct {
		Domain   string `json:"domain"`
		Port     int    `json:"port"`
		Endpoint string `json:"endpoint"`
	} `json:"searchWebAPI"`
	Echo struct {
		Port int `json:"port"`
	} `json:"echo"`
	Roles map[string]string `json:"roles"`
}

func LoadConfig(name string) (Config, error) {
	f, err := os.Open(name)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open config file: %v", err)
	}
	defer f.Close()

	var cfg Config
	err = json.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return Config{}, fmt.Errorf("failed to parse config file: %v", err)
	}

	cfg.Roles = reverseRolesMap(cfg.Roles)

	return cfg, nil
}

func reverseRolesMap(m map[string]string) map[string]string {
	reversed := make(map[string]string, len(m))
	for k, v := range m {
		reversed[v] = k
	}
	return reversed
}

func (cfg Config) EchoAddress() string {
	return fmt.Sprintf("localhost:%d", cfg.Echo.Port)
}
