package cidr

import (
	"bytes"
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// OutputFormatter defines methods for formatting network information
type OutputFormatter interface {
	FormatTable() string
	FormatJSON() ([]byte, error)
	FormatYAML() ([]byte, error)
}

// JSONFormatter handles JSON output formatting
type JSONFormatter struct {
	data *NetworkInfoOutput
}

// NewJSONFormatter creates a new JSON formatter for network info
func NewJSONFormatter(info *NetworkInfo) *JSONFormatter {
	return &JSONFormatter{
		data: info.ToOutput(),
	}
}

// Format returns the JSON representation of the network info
func (f *JSONFormatter) Format() ([]byte, error) {
	return json.MarshalIndent(f.data, "", "  ")
}

// YAMLFormatter handles YAML output formatting  
type YAMLFormatter struct {
	data *NetworkInfoOutput
}

// NewYAMLFormatter creates a new YAML formatter for network info
func NewYAMLFormatter(info *NetworkInfo) *YAMLFormatter {
	return &YAMLFormatter{
		data: info.ToOutput(),
	}
}

// Format returns the YAML representation of the network info
func (f *YAMLFormatter) Format() ([]byte, error) {
	return yaml.Marshal(f.data)
}

// TableFormatter handles table output formatting
type TableFormatter struct {
	info *NetworkInfo
}

// NewTableFormatter creates a new table formatter for network info
func NewTableFormatter(info *NetworkInfo) *TableFormatter {
	return &TableFormatter{
		info: info,
	}
}

// Format returns the table representation of the network info
func (f *TableFormatter) Format() string {
	var buf bytes.Buffer
	
	// This would contain the table formatting logic
	// For now, we'll return a simple string representation
	buf.WriteString(fmt.Sprintf("CIDR: %s\n", f.info.Network.String()))
	buf.WriteString(fmt.Sprintf("Base Address: %s\n", f.info.BaseAddress.String()))
	buf.WriteString(fmt.Sprintf("Total Addresses: %s\n", FormatBigInt(f.info.TotalAddresses)))
	
	return buf.String()
}

// FormatterFactory creates formatters based on output type
type FormatterFactory struct{}

// CreateJSONFormatter creates a JSON formatter
func (ff *FormatterFactory) CreateJSONFormatter(info *NetworkInfo) *JSONFormatter {
	return NewJSONFormatter(info)
}

// CreateYAMLFormatter creates a YAML formatter
func (ff *FormatterFactory) CreateYAMLFormatter(info *NetworkInfo) *YAMLFormatter {
	return NewYAMLFormatter(info)
}

// CreateTableFormatter creates a table formatter
func (ff *FormatterFactory) CreateTableFormatter(info *NetworkInfo) *TableFormatter {
	return NewTableFormatter(info)
}

// FormatAs provides a unified way to format network info in different formats
func (info *NetworkInfo) FormatAs(format string) ([]byte, error) {
	factory := &FormatterFactory{}
	
	switch format {
	case "json":
		formatter := factory.CreateJSONFormatter(info)
		return formatter.Format()
	case "yaml":
		formatter := factory.CreateYAMLFormatter(info)
		return formatter.Format()
	case "table":
		formatter := factory.CreateTableFormatter(info)
		return []byte(formatter.Format()), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
} 