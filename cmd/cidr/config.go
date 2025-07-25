package cidr

import (
	"fmt"
)

// ExplainConfig holds configuration for the explain command
type ExplainConfig struct {
	OutputFormat string
}

// Validate checks if the explain configuration is valid
func (c *ExplainConfig) Validate() error {
	validFormats := []string{"table", "json", "yaml"}
	for _, format := range validFormats {
		if c.OutputFormat == format {
			return nil
		}
	}
	return fmt.Errorf("invalid format '%s': supported formats are %v", c.OutputFormat, validFormats)
}

// ExpandConfig holds configuration for the expand command
type ExpandConfig struct {
	Limit   int
	OneLine bool
}

// Validate checks if the expand configuration is valid
func (c *ExpandConfig) Validate() error {
	if c.Limit < 0 {
		return fmt.Errorf("limit must be non-negative, got %d", c.Limit)
	}
	return nil
}

// CommandConfig holds common configuration across all CIDR commands
type CommandConfig struct {
	Debug   bool
	Verbose bool
}

// GlobalConfig combines all command configurations
type GlobalConfig struct {
	Command *CommandConfig
	Explain *ExplainConfig
	Expand  *ExpandConfig
}

// NewGlobalConfig creates a new global configuration with defaults
func NewGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		Command: &CommandConfig{
			Debug:   false,
			Verbose: false,
		},
		Explain: &ExplainConfig{
			OutputFormat: "table",
		},
		Expand: &ExpandConfig{
			Limit:   0,
			OneLine: false,
		},
	}
} 