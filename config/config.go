package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
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
	Log struct {
		Level   string `json:"level"`
		Path    string `json:"path"`
		Console bool   `json:"console"`
	}
	Roles   map[string]string `json:"roles"`
	RoleMap map[string]map[string]struct{}
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

	cfg.RoleMap = getRoleMap(cfg.Roles)
	log.Debug().Msgf("cfg.RoleMap: %+v", cfg.RoleMap)

	return cfg, nil
}

// getRoleMap converts a map of role strings to a nested map structure.
//
// It takes a map where the keys are role identifiers and the values are comma-separated
// strings of roles. It returns a map where each role identifier maps to another map,
// with each role as a key and an empty struct as the value.
//
// Parameters:
//   m (map[string]string): A map where keys are role identifiers and values are comma-separated roles.
//
// Returns:
//   map[string]map[string]struct{}: A nested map where each role identifier points to a map
//   with role keys and empty struct values.

func getRoleMap(m map[string]string) map[string]map[string]struct{} {
	// initialize nested map
	roleMap := make(map[string]map[string]struct{})
	for k := range m {
		roleMap[k] = make(map[string]struct{})
	}

	for k, v := range m {
		roles := strings.Split(v, ",")
		for _, role := range roles {
			role = strings.TrimSpace(role)
			if role != "" {
				roleMap[k][role] = struct{}{}
			}
		}
	}
	return roleMap
}

func (cfg Config) EchoAddress() string {
	return fmt.Sprintf("localhost:%d", cfg.Echo.Port)
}
