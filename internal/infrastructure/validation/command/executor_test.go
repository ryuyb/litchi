package command

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewExecutor(t *testing.T) {
	tests := []struct {
		name           string
		params         ExecutorParams
		expectedTimeout time.Duration
	}{
		{
			name:           "default timeout and logger",
			params:         ExecutorParams{},
			expectedTimeout: 5 * time.Minute,
		},
		{
			name: "custom timeout",
			params: ExecutorParams{
				Timeout: 30 * time.Second,
			},
			expectedTimeout: 30 * time.Second,
		},
		{
			name: "custom logger",
			params: ExecutorParams{
				Logger: zap.NewNop(),
			},
			expectedTimeout: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(tt.params)
			assert.NotNil(t, executor)
			assert.Equal(t, tt.expectedTimeout, executor.timeout)
		})
	}
}

func TestExecutor_Exec(t *testing.T) {
	executor := NewExecutor(ExecutorParams{
		Logger:  zap.NewNop(),
		Timeout: 10 * time.Second,
	})

	// Create temp directory for testing
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		cmd         string
		args        []string
		env         map[string]string
		timeout     int
		expectError bool
		errorMsg    string
	}{
		{
			name:    "successful command",
			cmd:     "echo",
			args:    []string{"hello"},
			env:     nil,
			timeout: 5,
		},
		{
			name:    "command with exit code",
			cmd:     "ls",
			args:    []string{"/nonexistent"},
			env:     nil,
			timeout: 5,
		},
		{
			name:        "empty command",
			cmd:         "",
			args:        []string{},
			env:         nil,
			timeout:     5,
			expectError: true,
			errorMsg:    "command cannot be empty",
		},
		{
			name:        "dangerous command with shell chars",
			cmd:         "echo;rm",
			args:        []string{},
			env:         nil,
			timeout:     5,
			expectError: true,
			errorMsg:    "command contains dangerous characters",
		},
		{
			name:        "dangerous command with pipe",
			cmd:         "cat|grep",
			args:        []string{},
			env:         nil,
			timeout:     5,
			expectError: true,
			errorMsg:    "command contains dangerous characters",
		},
		{
			name:    "command with environment variables",
			cmd:     "sh",
			args:    []string{"-c", "echo $TEST_VAR"},
			env:     map[string]string{"TEST_VAR": "test_value"},
			timeout: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := executor.Exec(ctx, tmpDir, tt.cmd, tt.args, tt.env, tt.timeout)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}
}

func TestExecutor_ExecWithTimeout(t *testing.T) {
	executor := NewExecutor(ExecutorParams{
		Logger:  zap.NewNop(),
		Timeout: 5 * time.Second,
	})

	tmpDir := t.TempDir()

	// Test timeout with a long-running command
	ctx := context.Background()
	result, err := executor.Exec(ctx, tmpDir, "sleep", []string{"10"}, nil, 1) // 1 second timeout

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timeout")
	assert.Nil(t, result)
}

func TestExecutor_ExecWithOutput(t *testing.T) {
	executor := NewExecutor(ExecutorParams{
		Logger:  zap.NewNop(),
		Timeout: 10 * time.Second,
	})

	tmpDir := t.TempDir()

	tests := []struct {
		name            string
		cmd             string
		args            []string
		expectedContent string
	}{
		{
			name:            "echo output",
			cmd:             "echo",
			args:            []string{"test output"},
			expectedContent: "test output",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			output, duration, err := executor.ExecWithOutput(ctx, tmpDir, tt.cmd, tt.args, nil, 5)

			require.NoError(t, err)
			assert.Contains(t, output, tt.expectedContent)
			assert.Greater(t, duration, int64(0))
		})
	}
}

func TestExecutor_CheckFileExists(t *testing.T) {
	executor := NewExecutor(ExecutorParams{
		Logger: zap.NewNop(),
	})

	tmpDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test"), 0644))

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "existing file",
			filename: "test.txt",
			expected: true,
		},
		{
			name:     "non-existing file",
			filename: "nonexistent.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := executor.CheckFileExists(tmpDir, tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecutor_FindFiles(t *testing.T) {
	executor := NewExecutor(ExecutorParams{
		Logger: zap.NewNop(),
	})

	tmpDir := t.TempDir()

	// Create test files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test1.go"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test2.go"), []byte("test"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644))

	// Create nested directory with file
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "test3.go"), []byte("test"), 0644))

	tests := []struct {
		name          string
		pattern       string
		expectedCount int
	}{
		{
			name:          "find go files",
			pattern:       "*.go",
			expectedCount: 3,
		},
		{
			name:          "find txt files",
			pattern:       "*.txt",
			expectedCount: 1,
		},
		{
			name:          "find non-matching pattern",
			pattern:       "*.md",
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files, err := executor.FindFiles(tmpDir, tt.pattern)
			require.NoError(t, err)
			assert.Len(t, files, tt.expectedCount)
		})
	}
}

func TestExecutor_FindFiles_SkipsDirectories(t *testing.T) {
	executor := NewExecutor(ExecutorParams{
		Logger: zap.NewNop(),
	})

	tmpDir := t.TempDir()

	// Create files in skipped directories
	nodeModules := filepath.Join(tmpDir, "node_modules")
	require.NoError(t, os.Mkdir(nodeModules, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(nodeModules, "skip.go"), []byte("test"), 0644))

	vendor := filepath.Join(tmpDir, "vendor")
	require.NoError(t, os.Mkdir(vendor, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(vendor, "skip.go"), []byte("test"), 0644))

	// Create file in normal directory
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "include.go"), []byte("test"), 0644))

	files, err := executor.FindFiles(tmpDir, "*.go")
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Contains(t, files, "include.go")
}

func TestExecutor_ReadFile(t *testing.T) {
	executor := NewExecutor(ExecutorParams{
		Logger: zap.NewNop(),
	})

	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("test content")
	require.NoError(t, os.WriteFile(testFile, testContent, 0644))

	// Test successful read
	content, err := executor.ReadFile(tmpDir, "test.txt")
	require.NoError(t, err)
	assert.Equal(t, testContent, content)

	// Test non-existing file
	_, err = executor.ReadFile(tmpDir, "nonexistent.txt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestExecutor_SetTimeout(t *testing.T) {
	executor := NewExecutor(ExecutorParams{
		Logger:  zap.NewNop(),
		Timeout: 5 * time.Minute,
	})

	newTimeout := 10 * time.Minute
	executor.SetTimeout(newTimeout)
	assert.Equal(t, newTimeout, executor.timeout)
}

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid command",
			cmd:         "echo",
			expectError: false,
		},
		{
			name:        "empty command",
			cmd:         "",
			expectError: true,
			errorMsg:    "command cannot be empty",
		},
		{
			name:        "command with semicolon",
			cmd:         "echo;rm",
			expectError: true,
			errorMsg:    "command contains dangerous characters",
		},
		{
			name:        "command with pipe",
			cmd:         "cat|grep",
			expectError: true,
			errorMsg:    "command contains dangerous characters",
		},
		{
			name:        "command with backtick",
			cmd:         "echo`id`",
			expectError: true,
			errorMsg:    "command contains dangerous characters",
		},
		{
			name:        "command with dollar sign",
			cmd:         "echo$(id)",
			expectError: true,
			errorMsg:    "command contains dangerous characters",
		},
		{
			name:        "command with redirect",
			cmd:         "cat>file",
			expectError: true,
			errorMsg:    "command contains dangerous characters",
		},
		{
			name:        "command with backslash",
			cmd:         "echo\\test",
			expectError: true,
			errorMsg:    "command contains dangerous characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCommand(tt.cmd)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}