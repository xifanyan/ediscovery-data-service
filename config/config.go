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
}

func LoadConfig(name string) (Config, error) {
	// exePath, _ := os.Executable()
	// path := filepath.Join(filepath.Dir(exePath), name)

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

	return cfg, nil
}

func (cfg Config) EchoAddress() string {
	return fmt.Sprintf("localhost:%d", cfg.Echo.Port)
}
