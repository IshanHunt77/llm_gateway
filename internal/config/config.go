package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Provider struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"-"`
}

type Config struct {
	GatewayPort     string     `json:"gateway_port"`
	Providers       []Provider `json:"providers"`
	DefaultProvider string     `json:"default_provider"`
}

func Load() (*Config, error) {
	var cfg Config
	jsonblob, err := os.ReadFile("config.json")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonblob, &cfg)
	if err != nil {
		return nil, err
	}
	for i := range cfg.Providers {
		envName := strings.ToUpper(cfg.Providers[i].Name)+"_API_KEY"
		cfg.Providers[i].APIKey = os.Getenv(envName)
	}
	return &cfg, nil
}

func (c *Config) DefaultProviderURL() (string, error) {
	for _, prov := range c.Providers {
		if prov.Name == c.DefaultProvider {
			return prov.BaseURL, nil
		}

	}
	return "", fmt.Errorf("provider %q not found", c.DefaultProvider)
}
