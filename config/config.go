// Package config provides config schemas, loading and parsing for hops
package config

import (
	"fmt"
	"path/filepath"

	"github.com/ilyakaznacheev/cleanenv"
)

const ConfigDirName = ".hiphops"

type (
	HopsConf struct {
		Dev        bool       `yaml:"dev" env:"HIPHOPS_DEV"`
		Runner     RunnerConf `yaml:"runner" env-prefix:"HIPHOPS_RUNNER_"`
		hiphopsDir string
		tag        string
	}

	RunnerConf struct {
		Serve    bool   `yaml:"serve" env:"SERVE"`
		NATSConf string `yaml:"nats_config" env:"NATS_CONFIG"`
		Local    bool   `yaml:"local" env:"LOCAL"` // TODO: Check we actually use/need this
		// TODO: Add LogLevel as separate config
	}
)

func LoadConfig(hiphopsDir string, tag string) (*HopsConf, error) {
	h := &HopsConf{
		hiphopsDir: hiphopsDir,
		tag:        tag,
	}

	h.Runner = RunnerConf{
		NATSConf: h.NATSConfigPath(),
	}

	if err := cleanenv.ReadConfig(h.BaseConfigPath(), h); err != nil {
		return nil, err
	}

	if tag == "" {
		return h, nil
	}

	err := cleanenv.ReadConfig(h.ConfigPath(), h)

	return h, err
}

func (h *HopsConf) BaseConfigPath() string {
	return filepath.Join(h.ConfigDir(), "config.yaml")
}

func (h *HopsConf) ConfigDir() string {
	return filepath.Join(h.hiphopsDir, ".hiphops")
}

func (h *HopsConf) ConfigPath() string {
	return filepath.Join(h.ConfigDir(), fmt.Sprintf("config.%s.yaml", h.tag))
}

func (h *HopsConf) NATSConfigPath() string {
	return filepath.Join(h.ConfigDir(), "nats.conf")
}

func (h *HopsConf) FlowsPath() string {
	return filepath.Join(h.hiphopsDir, "flows")
}
