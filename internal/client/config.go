package client

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type HostConfig struct {
	Host string `yaml:"host"`
	User string `yaml:"user"`
	Key  string `yaml:"key"`
	Port int    `yaml:"port,omitempty"`
}

type Config struct {
	Hosts map[string]HostConfig `yaml:"hosts"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentd")
}

func configPath() string {
	return filepath.Join(configDir(), "hosts.yaml")
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(configPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Hosts: make(map[string]HostConfig)}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Hosts == nil {
		cfg.Hosts = make(map[string]HostConfig)
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0600)
}

func (c *Config) AddHost(name string, host HostConfig) {
	c.Hosts[name] = host
}

func (c *Config) RemoveHost(name string) {
	delete(c.Hosts, name)
}

func (c *Config) GetHost(name string) (HostConfig, bool) {
	h, ok := c.Hosts[name]
	return h, ok
}
