package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	type testCase struct {
		name             string
		tag              string
		configFiles      map[string][]byte
		envVars          map[string]string
		wantError        bool
		expectedHopsConf Config
	}

	tests := []testCase{
		{
			name: "Base config loading",
			configFiles: map[string][]byte{
				"": []byte(`
dev: true
`),
			},
			expectedHopsConf: Config{
				Dev: true,
			},
		},
		{
			name: "Env var precedence",
			configFiles: map[string][]byte{
				"": []byte(`
dev: true
`),
			},
			envVars: map[string]string{
				"HIPHOPS_DEV": "false",
			},
			expectedHopsConf: Config{
				Dev: false,
			},
		},
		{
			name: "Tag overlays",
			tag:  "dev",
			configFiles: map[string][]byte{
				"": []byte(`
dev: true
runner:
  local: false
`),
				"dev": []byte(`
runner:
  local: true
`),
			},
			expectedHopsConf: Config{
				Dev: true,
				Runner: RunnerConf{
					Local: true,
				},
			},
		},
		{
			name: "Bad config",
			configFiles: map[string][]byte{
				"": []byte(`
dev: "not a boolean val"
runner:
  local: false
`),
			},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hopsDir := setupHopsDir(t, tc.configFiles)

			cleanEnvVars(t, []string{
				"HIPHOPS_DEV",
				"HIPHOPS_RUNNER_NATS_CONFIG",
				"HIPHOPS_RUNNER_DATA_DIR",
				"HIPHOPS_RUNNER_LOCAL",
			})

			for name, value := range tc.envVars {
				t.Setenv(name, value)
			}

			hopsConf, err := LoadConfig(hopsDir, tc.tag)
			if tc.wantError {
				assert.Error(t, err, "Config loading should return an error")
				return
			}

			tc.expectedHopsConf.tag = tc.tag
			tc.expectedHopsConf.hiphopsDir = hopsDir
			if tc.expectedHopsConf.Runner.NATSConf == "" {
				tc.expectedHopsConf.Runner.NATSConf = filepath.Join(hopsDir, "hiphops", "nats.conf")
			}

			assert.NoError(t, err, "Config should load without error")
			assert.Equal(t, tc.expectedHopsConf, *hopsConf, "Config should have correct values")
		})
	}
}

// setupHopsDir is a test helper to create a hops directory structure with
// populated config files. Configs are given as a map of tags and their contents
func setupHopsDir(t *testing.T, configs map[string][]byte) string {
	hopsDir := t.TempDir()
	configDir := filepath.Join(hopsDir, "hiphops")
	err := os.Mkdir(configDir, 0744)
	require.NoError(t, err) // Abort the test if setup fails

	for tag, content := range configs {
		configFile := "config.yaml"
		if tag != "" {
			configFile = fmt.Sprintf("config.%s.yaml", tag)
		}

		path := filepath.Join(configDir, configFile)
		err := os.WriteFile(path, content, 0644)
		require.NoError(t, err) // Abort the test if setup fails
	}

	return hopsDir
}

func cleanEnvVars(t *testing.T, envVars []string) {
	for _, env := range envVars {
		err := os.Unsetenv(env)
		require.NoError(t, err, "Test setup: Unable to clean env vars")
	}
}
