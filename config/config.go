// Package config provides config schemas, loading and parsing for hops
package config

import (
	"fmt"
	"path/filepath"

	"github.com/ilyakaznacheev/cleanenv"
)

const ConfigDirName = ".hiphops"

type (
	Config struct {
		Dev        bool       `yaml:"dev" env:"HIPHOPS_DEV"`
		Runner     RunnerConf `yaml:"runner" env-prefix:"HIPHOPS_RUNNER_"`
		hiphopsDir string
		tag        string
	}

	RunnerConf struct {
		Serve    bool   `yaml:"serve" env:"SERVE"` // TODO: I think this can now be removed?
		NATSConf string `yaml:"nats_config" env:"NATS_CONFIG"`
		DataDir  string `yaml:"data_dir" env:"DATA_DIR"`
		Local    bool   `yaml:"local" env:"LOCAL"` // TODO: Check we actually use/need this
		// TODO: Add LogLevel as separate config
	}
)

func NewConfig(hiphopsDir string, tag string) *Config {
	return &Config{
		hiphopsDir: hiphopsDir,
		tag:        tag,
	}
}

func LoadConfig(hiphopsDir string, tag string) (*Config, error) {
	c := &Config{
		hiphopsDir: hiphopsDir,
		tag:        tag,
	}

	c.Runner = RunnerConf{
		NATSConf: c.NATSConfigPath(),
	}

	if err := cleanenv.ReadConfig(c.BaseConfigPath(), c); err != nil {
		return nil, err
	}

	if tag == "" {
		return c, nil
	}

	err := cleanenv.ReadConfig(c.ConfigPath(), c)

	return c, err
}

func (c *Config) BaseConfigPath() string {
	return filepath.Join(c.ConfigDirPath(), "config.yaml")
}

func (c *Config) ConfigDirPath() string {
	return filepath.Join(c.hiphopsDir, "hiphops")
}

func (c *Config) ConfigPath() string {
	if c.tag == "" {
		return ""
	}

	return filepath.Join(c.ConfigDirPath(), fmt.Sprintf("config.%s.yaml", c.tag))
}

func (c *Config) DockerComposePath() string {
	return filepath.Join(c.LocalDirPath(), "docker-compose.yaml")
}

func (c *Config) FlowsPath() string {
	return filepath.Join(c.hiphopsDir, "flows")
}

func (c *Config) LocalDirPath() string {
	return filepath.Join(c.hiphopsDir, ConfigDirName)
}

func (c *Config) NATSConfigPath() string {
	return filepath.Join(c.ConfigDirPath(), "nats.conf")
}
