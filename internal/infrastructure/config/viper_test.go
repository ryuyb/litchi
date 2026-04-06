package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getProjectRoot returns the project root directory.
func getProjectRoot(t *testing.T) string {
	// Get current working directory
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Navigate up to find project root (contains go.mod)
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root")
		}
		dir = parent
	}
}

func TestDetectEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected Environment
	}{
		{
			name:     "no env vars - default to dev",
			envVars:  map[string]string{},
			expected: EnvDev,
		},
		{
			name: "LITCHI_ENV takes precedence",
			envVars: map[string]string{
				"LITCHI_ENV": "prod",
				"GO_ENV":     "uat",
				"ENV":        "dev",
			},
			expected: EnvProd,
		},
		{
			name: "GO_ENV is second priority",
			envVars: map[string]string{
				"GO_ENV": "uat",
				"ENV":    "dev",
			},
			expected: EnvUAT,
		},
		{
			name: "ENV is third priority",
			envVars: map[string]string{
				"ENV": "prod",
			},
			expected: EnvProd,
		},
		{
			name: "case insensitive",
			envVars: map[string]string{
				"LITCHI_ENV": "PROD",
			},
			expected: EnvProd,
		},
		{
			name: "custom environment value allowed",
			envVars: map[string]string{
				"LITCHI_ENV": "staging",
			},
			expected: Environment("staging"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all env vars first
			os.Unsetenv("LITCHI_ENV")
			os.Unsetenv("GO_ENV")
			os.Unsetenv("ENV")

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result := detectEnvironment("")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewConfigWithOptions_EnvironmentField(t *testing.T) {
	projectRoot := getProjectRoot(t)

	// Set required env vars for validation
	os.Setenv("DB_PASSWORD", "testpassword")
	os.Setenv("GITHUB_TOKEN", "testtoken")
	os.Setenv("GITHUB_WEBHOOK_SECRET", "testsecret")
	os.Setenv("LITCHI_CONFIG_DIR", filepath.Join(projectRoot, "config"))
	defer func() {
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("GITHUB_TOKEN")
		os.Unsetenv("GITHUB_WEBHOOK_SECRET")
		os.Unsetenv("LITCHI_CONFIG_DIR")
	}()

	cfg, err := NewConfigWithOptions(LoadOptions{Env: EnvDev})
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, EnvDev, cfg.Environment())
}

func TestNewConfigWithOptions_MissingEnvConfig(t *testing.T) {
	projectRoot := getProjectRoot(t)

	// Set required env vars for validation
	os.Setenv("DB_PASSWORD", "testpassword")
	os.Setenv("GITHUB_TOKEN", "testtoken")
	os.Setenv("GITHUB_WEBHOOK_SECRET", "testsecret")
	os.Setenv("LITCHI_CONFIG_DIR", filepath.Join(projectRoot, "config"))
	defer func() {
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("GITHUB_TOKEN")
		os.Unsetenv("GITHUB_WEBHOOK_SECRET")
		os.Unsetenv("LITCHI_CONFIG_DIR")
	}()

	// Test with an environment that doesn't have a config file
	cfg, err := NewConfigWithOptions(LoadOptions{Env: Environment("nonexistent")})
	require.NoError(t, err) // Should not error, just skip the env config
	require.NotNil(t, cfg)

	assert.Equal(t, Environment("nonexistent"), cfg.Environment())
}