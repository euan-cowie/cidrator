package mtu

import (
	"encoding/json"
	"os"
)

func writePrettyJSON(v any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func writeJSONLine(v any) error {
	return json.NewEncoder(os.Stdout).Encode(v)
}
