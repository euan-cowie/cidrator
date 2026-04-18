package dns

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	internaldns "github.com/euan-cowie/cidrator/internal/dns"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newLookupTestCommand(out *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{Use: "lookup <domain>", RunE: runLookup}
	cmd.SetOut(out)
	cmd.Flags().StringP("type", "t", "A", "DNS record type")
	cmd.Flags().StringP("format", "f", "table", "Output format")
	cmd.Flags().StringP("server", "s", "", "DNS server")
	cmd.Flags().Duration("timeout", 5*time.Second, "Query timeout")
	return cmd
}

func newReverseTestCommand(out *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{Use: "reverse <ip>", RunE: runReverse}
	cmd.SetOut(out)
	cmd.Flags().StringP("format", "f", "table", "Output format")
	cmd.Flags().Duration("timeout", 5*time.Second, "Query timeout")
	return cmd
}

func TestOutputLookupResult(t *testing.T) {
	result := &internaldns.DNSResult{
		Domain:    "example.com",
		QueryType: "MX",
		QueryTime: 1500 * time.Millisecond,
		Server:    "1.1.1.1",
		Records: []internaldns.DNSRecord{
			{Type: "MX", Priority: 10, Value: "mail1.example.com"},
			{Type: "A", Value: "192.0.2.1"},
		},
	}

	t.Run("json", func(t *testing.T) {
		var out bytes.Buffer
		if err := outputLookupResult(&out, result, "json"); err != nil {
			t.Fatalf("outputLookupResult returned error: %v", err)
		}

		var payload struct {
			Domain      string                  `json:"domain"`
			QueryType   string                  `json:"query_type"`
			QueryTimeMS int64                   `json:"query_time_ms"`
			Server      string                  `json:"server"`
			Records     []internaldns.DNSRecord `json:"records"`
		}
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("invalid JSON output: %v", err)
		}
		if payload.Domain != "example.com" || payload.QueryType != "MX" || payload.QueryTimeMS != 1500 {
			t.Fatalf("unexpected JSON payload: %+v", payload)
		}
	})

	t.Run("yaml", func(t *testing.T) {
		var out bytes.Buffer
		if err := outputLookupResult(&out, result, "yaml"); err != nil {
			t.Fatalf("outputLookupResult returned error: %v", err)
		}

		var payload map[string]any
		if err := yaml.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("invalid YAML output: %v", err)
		}
		if payload["domain"] != "example.com" || payload["query_type"] != "MX" {
			t.Fatalf("unexpected YAML payload: %#v", payload)
		}
	})

	t.Run("table", func(t *testing.T) {
		var out bytes.Buffer
		if err := outputLookupResult(&out, result, "table"); err != nil {
			t.Fatalf("outputLookupResult returned error: %v", err)
		}

		output := out.String()
		expected := []string{
			"Domain: example.com",
			"Query Type: MX",
			"Server: 1.1.1.1",
			"TYPE",
			"PRIORITY",
			"mail1.example.com",
			"192.0.2.1",
		}
		for _, fragment := range expected {
			if !strings.Contains(output, fragment) {
				t.Fatalf("expected table output to contain %q, got %q", fragment, output)
			}
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		var out bytes.Buffer
		err := outputLookupResult(&out, result, "xml")
		if err == nil || !strings.Contains(err.Error(), "unsupported output format") {
			t.Fatalf("expected unsupported format error, got %v", err)
		}
	})
}

func TestOutputLookupTableNoRecords(t *testing.T) {
	var out bytes.Buffer

	outputLookupTable(&out, &internaldns.DNSResult{
		Domain:    "example.com",
		QueryType: "A",
		QueryTime: 25 * time.Millisecond,
	})

	if !strings.Contains(out.String(), "No records found.") {
		t.Fatalf("expected no records message, got %q", out.String())
	}
}

func TestRunLookupUsesFlagsAndWriter(t *testing.T) {
	original := dnsLookup
	t.Cleanup(func() { dnsLookup = original })

	var gotDomain string
	var gotOpts internaldns.LookupOptions
	dnsLookup = func(domain string, opts internaldns.LookupOptions) (*internaldns.DNSResult, error) {
		gotDomain = domain
		gotOpts = opts
		return &internaldns.DNSResult{
			Domain:    domain,
			QueryType: opts.RecordType,
			Server:    opts.Server,
			QueryTime: 10 * time.Millisecond,
			Records:   []internaldns.DNSRecord{{Type: opts.RecordType, Value: "192.0.2.10"}},
		}, nil
	}

	var out bytes.Buffer
	cmd := newLookupTestCommand(&out)
	cmd.SetArgs([]string{"example.com", "--type", "mx", "--format", "json", "--server", "8.8.8.8", "--timeout", "2s"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("lookup command failed: %v", err)
	}

	if gotDomain != "example.com" {
		t.Fatalf("unexpected domain: %q", gotDomain)
	}
	if gotOpts.RecordType != "MX" || gotOpts.Server != "8.8.8.8" || gotOpts.Timeout != 2*time.Second {
		t.Fatalf("unexpected lookup options: %+v", gotOpts)
	}

	var payload map[string]any
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("lookup command produced invalid JSON: %v", err)
	}
	if payload["domain"] != "example.com" || payload["query_type"] != "MX" {
		t.Fatalf("unexpected command output: %#v", payload)
	}
}

func TestOutputReverseResult(t *testing.T) {
	result := &internaldns.ReverseResult{
		IP:        "192.0.2.10",
		Hostnames: []string{"host1.example.com", "host2.example.com"},
		QueryTime: 1200 * time.Millisecond,
	}

	t.Run("json", func(t *testing.T) {
		var out bytes.Buffer
		if err := outputReverseResult(&out, result, "json"); err != nil {
			t.Fatalf("outputReverseResult returned error: %v", err)
		}

		var payload struct {
			IP          string   `json:"ip"`
			Hostnames   []string `json:"hostnames"`
			QueryTimeMS int64    `json:"query_time_ms"`
		}
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("invalid JSON output: %v", err)
		}
		if payload.IP != "192.0.2.10" || payload.QueryTimeMS != 1200 {
			t.Fatalf("unexpected JSON payload: %+v", payload)
		}
	})

	t.Run("table", func(t *testing.T) {
		var out bytes.Buffer
		if err := outputReverseResult(&out, result, "table"); err != nil {
			t.Fatalf("outputReverseResult returned error: %v", err)
		}

		output := out.String()
		expected := []string{"IP: 192.0.2.10", "Hostnames:", "host1.example.com", "host2.example.com"}
		for _, fragment := range expected {
			if !strings.Contains(output, fragment) {
				t.Fatalf("expected reverse table output to contain %q, got %q", fragment, output)
			}
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		var out bytes.Buffer
		err := outputReverseResult(&out, result, "xml")
		if err == nil || !strings.Contains(err.Error(), "unsupported output format") {
			t.Fatalf("expected unsupported format error, got %v", err)
		}
	})
}

func TestOutputReverseTableNoRecords(t *testing.T) {
	var out bytes.Buffer

	outputReverseTable(&out, &internaldns.ReverseResult{
		IP:        "192.0.2.10",
		QueryTime: 25 * time.Millisecond,
	})

	if !strings.Contains(out.String(), "No PTR records found.") {
		t.Fatalf("expected no PTR message, got %q", out.String())
	}
}

func TestRunReverseUsesFlagsAndWriter(t *testing.T) {
	original := dnsReverseLookup
	t.Cleanup(func() { dnsReverseLookup = original })

	var gotIP string
	var gotTimeout time.Duration
	dnsReverseLookup = func(ip string, timeout time.Duration) (*internaldns.ReverseResult, error) {
		gotIP = ip
		gotTimeout = timeout
		return &internaldns.ReverseResult{
			IP:        ip,
			Hostnames: []string{"host.example.com"},
			QueryTime: 5 * time.Millisecond,
		}, nil
	}

	var out bytes.Buffer
	cmd := newReverseTestCommand(&out)
	cmd.SetArgs([]string{"192.0.2.10", "--format", "yaml", "--timeout", "1500ms"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("reverse command failed: %v", err)
	}

	if gotIP != "192.0.2.10" || gotTimeout != 1500*time.Millisecond {
		t.Fatalf("unexpected reverse lookup inputs: ip=%q timeout=%v", gotIP, gotTimeout)
	}
	if !strings.Contains(out.String(), "host.example.com") {
		t.Fatalf("expected reverse command output to contain hostname, got %q", out.String())
	}
}
