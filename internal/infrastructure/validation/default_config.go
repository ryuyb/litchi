package validation

import (
	"github.com/ryuyb/litchi/internal/domain/valueobject"
)

// DefaultConfigGenerator generates default validation configs.
type DefaultConfigGenerator struct{}

// NewDefaultConfigGenerator creates a new default config generator.
func NewDefaultConfigGenerator() *DefaultConfigGenerator {
	return &DefaultConfigGenerator{}
}

// Generate generates validation configuration from detected project.
func (g *DefaultConfigGenerator) Generate(detected *valueobject.DetectedProject) *valueobject.ExecutionValidationConfig {
	if detected == nil {
		return g.GenerateForLanguage("unknown")
	}

	config := &valueobject.ExecutionValidationConfig{
		Enabled: true,
		Formatting: valueobject.FormattingConfig{
			Enabled:         true,
			Tools:           []valueobject.ToolCommand{},
			FailureStrategy: valueobject.AutoFix,
		},
		Linting: valueobject.LintingConfig{
			Enabled:         true,
			Tools:           []valueobject.ToolCommand{},
			FailureStrategy: valueobject.AutoFix,
			AutoFix:         true,
		},
		Testing: valueobject.TestingConfig{
			Enabled:         true,
			Command:         valueobject.ToolCommand{},
			FailureStrategy: valueobject.FailFast,
			NoTestsStrategy: valueobject.WarnNoTests,
		},
		AutoDetection: valueobject.AutoDetectionConfig{
			Enabled:         true,
			Mode:            valueobject.AutoDetectFull,
			DetectedProject: detected,
		},
	}

	// Extract tools from detected project
	for _, tool := range detected.DetectedTools {
		switch tool.Type {
		case valueobject.ToolTypeFormatter:
			config.Formatting.Tools = append(config.Formatting.Tools, tool.RecommendedCommand)
		case valueobject.ToolTypeLinter:
			config.Linting.Tools = append(config.Linting.Tools, tool.RecommendedCommand)
		case valueobject.ToolTypeTester:
			config.Testing.Command = tool.RecommendedCommand
		}
	}

	return config
}

// GenerateForLanguage generates default config for a specific language.
func (g *DefaultConfigGenerator) GenerateForLanguage(language string) *valueobject.ExecutionValidationConfig {
	config := &valueobject.ExecutionValidationConfig{
		Enabled: true,
		Formatting: valueobject.FormattingConfig{
			Enabled:         true,
			FailureStrategy: valueobject.AutoFix,
		},
		Linting: valueobject.LintingConfig{
			Enabled:         true,
			FailureStrategy: valueobject.AutoFix,
			AutoFix:         true,
		},
		Testing: valueobject.TestingConfig{
			Enabled:         true,
			FailureStrategy: valueobject.FailFast,
			NoTestsStrategy: valueobject.WarnNoTests,
		},
		AutoDetection: valueobject.AutoDetectionConfig{
			Enabled: true,
			Mode:    valueobject.AutoDetectFull,
		},
	}

	switch language {
	case "go":
		config.Formatting.Tools = []valueobject.ToolCommand{
			valueobject.NewToolCommand("gofmt", "gofmt", []string{"-w", "."}, 60),
		}
		config.Linting.Tools = []valueobject.ToolCommand{
			valueobject.NewToolCommand("golangci-lint", "golangci-lint", []string{"run", "--fix"}, 120),
		}
		config.Testing.Command = valueobject.NewToolCommand("go test", "go", []string{"test", "-v", "./..."}, 300)

	case "nodejs", "javascript", "typescript":
		config.Formatting.Tools = []valueobject.ToolCommand{
			valueobject.NewToolCommand("prettier", "npx", []string{"prettier", "--write", "."}, 120),
		}
		config.Linting.Tools = []valueobject.ToolCommand{
			valueobject.NewToolCommand("eslint", "npx", []string{"eslint", "--fix", "."}, 120),
		}
		config.Testing.Command = valueobject.NewToolCommand("jest", "npx", []string{"jest", "--passWithNoTests"}, 300)

	case "python":
		config.Formatting.Tools = []valueobject.ToolCommand{
			valueobject.NewToolCommand("black", "black", []string{"."}, 120),
		}
		config.Linting.Tools = []valueobject.ToolCommand{
			valueobject.NewToolCommand("ruff", "ruff", []string{"check", "--fix", "."}, 120),
		}
		config.Testing.Command = valueobject.NewToolCommand("pytest", "pytest", []string{}, 300)

	case "rust":
		config.Formatting.Tools = []valueobject.ToolCommand{
			valueobject.NewToolCommand("cargo fmt", "cargo", []string{"fmt"}, 120),
		}
		config.Linting.Tools = []valueobject.ToolCommand{
			valueobject.NewToolCommand("cargo clippy", "cargo", []string{"clippy", "--fix", "--allow-dirty"}, 180),
		}
		config.Testing.Command = valueobject.NewToolCommand("cargo test", "cargo", []string{"test"}, 300)

	default:
		// Disable validation for unknown languages
		config.Enabled = false
		config.Formatting.Enabled = false
		config.Linting.Enabled = false
		config.Testing.Enabled = false
	}

	return config
}