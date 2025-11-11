package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr string   `yaml:"listen_addr" json:"listen_addr"`
	MetaDSN    string   `yaml:"meta_dsn" json:"meta_dsn"`
	Storages   []string `yaml:"storages" json:"storages"`
}

// Load читает YAML-конфигурацию, применяет ENV-переопределения и возвращает актуальную структуру.
func Load() (*Config, error) {
	path := getenv("CONFIG_PATH", "./config.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}

	// ENV override
	if v := os.Getenv("LISTEN_ADDR"); v != "" {
		c.ListenAddr = v
	}
	if v := os.Getenv("META_DSN"); v != "" {
		c.MetaDSN = v
	}
	if v := os.Getenv("STORAGES"); v != "" {
		c.Storages = splitComma(v)
	}

	return &c, nil
}

func splitComma(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}

	return out
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}

	return def
}
