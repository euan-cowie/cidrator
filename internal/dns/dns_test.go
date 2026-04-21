package dns

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type fakeDNSResolver struct {
	lookupIPFunc    func(ctx context.Context, network, host string) ([]net.IP, error)
	lookupMXFunc    func(ctx context.Context, name string) ([]*net.MX, error)
	lookupTXTFunc   func(ctx context.Context, name string) ([]string, error)
	lookupCNAMEFunc func(ctx context.Context, host string) (string, error)
	lookupNSFunc    func(ctx context.Context, name string) ([]*net.NS, error)
}

func (f fakeDNSResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	if f.lookupIPFunc == nil {
		return nil, nil
	}
	return f.lookupIPFunc(ctx, network, host)
}

func (f fakeDNSResolver) LookupMX(ctx context.Context, name string) ([]*net.MX, error) {
	if f.lookupMXFunc == nil {
		return nil, nil
	}
	return f.lookupMXFunc(ctx, name)
}

func (f fakeDNSResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	if f.lookupTXTFunc == nil {
		return nil, nil
	}
	return f.lookupTXTFunc(ctx, name)
}

func (f fakeDNSResolver) LookupCNAME(ctx context.Context, host string) (string, error) {
	if f.lookupCNAMEFunc == nil {
		return "", nil
	}
	return f.lookupCNAMEFunc(ctx, host)
}

func (f fakeDNSResolver) LookupNS(ctx context.Context, name string) ([]*net.NS, error) {
	if f.lookupNSFunc == nil {
		return nil, nil
	}
	return f.lookupNSFunc(ctx, name)
}

type fakeReverseResolver struct {
	lookupAddrFunc func(ctx context.Context, addr string) ([]string, error)
}

func (f fakeReverseResolver) LookupAddr(ctx context.Context, addr string) ([]string, error) {
	return f.lookupAddrFunc(ctx, addr)
}

func TestDefaultLookupOptions(t *testing.T) {
	opts := DefaultLookupOptions()

	if opts.RecordType != RecordTypeA {
		t.Fatalf("expected default record type %q, got %q", RecordTypeA, opts.RecordType)
	}
	if opts.Timeout != 5*time.Second {
		t.Fatalf("expected default timeout 5s, got %v", opts.Timeout)
	}
}

func TestDNSResultSerialization(t *testing.T) {
	result := &DNSResult{
		Domain:    "example.com",
		QueryType: RecordTypeMX,
		QueryTime: 1500 * time.Millisecond,
		Server:    "1.1.1.1",
		Records: []DNSRecord{
			{Type: RecordTypeMX, Value: "mail.example.com", Priority: 10},
		},
	}

	jsonOutput, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON returned error: %v", err)
	}

	var jsonPayload map[string]any
	if err := json.Unmarshal([]byte(jsonOutput), &jsonPayload); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if jsonPayload["domain"] != "example.com" || jsonPayload["query_time_ms"] != float64(1500) {
		t.Fatalf("unexpected JSON payload: %#v", jsonPayload)
	}

	yamlOutput, err := result.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML returned error: %v", err)
	}

	var yamlPayload map[string]any
	if err := yaml.Unmarshal([]byte(yamlOutput), &yamlPayload); err != nil {
		t.Fatalf("invalid YAML output: %v", err)
	}
	if yamlPayload["server"] != "1.1.1.1" || yamlPayload["query_type"] != "MX" {
		t.Fatalf("unexpected YAML payload: %#v", yamlPayload)
	}
}

func TestReverseResultSerialization(t *testing.T) {
	result := &ReverseResult{
		IP:        "192.0.2.10",
		Hostnames: []string{"host.example.com"},
		QueryTime: 1200 * time.Millisecond,
	}

	jsonOutput, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON returned error: %v", err)
	}

	var jsonPayload map[string]any
	if err := json.Unmarshal([]byte(jsonOutput), &jsonPayload); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if jsonPayload["ip"] != "192.0.2.10" || jsonPayload["query_time_ms"] != float64(1200) {
		t.Fatalf("unexpected JSON payload: %#v", jsonPayload)
	}

	yamlOutput, err := result.ToYAML()
	if err != nil {
		t.Fatalf("ToYAML returned error: %v", err)
	}

	var yamlPayload map[string]any
	if err := yaml.Unmarshal([]byte(yamlOutput), &yamlPayload); err != nil {
		t.Fatalf("invalid YAML output: %v", err)
	}
	if yamlPayload["ip"] != "192.0.2.10" {
		t.Fatalf("unexpected YAML payload: %#v", yamlPayload)
	}
}

func TestLookupValidation(t *testing.T) {
	_, err := Lookup("", DefaultLookupOptions())
	if err == nil {
		t.Fatal("expected empty domain error")
	}
	if !errors.Is(err, ErrEmptyDomain) {
		t.Fatalf("expected ErrEmptyDomain, got %v", err)
	}

	_, err = Lookup(" example.com. ", LookupOptions{
		RecordType: "SRV",
		Timeout:    time.Second,
	})
	if err == nil {
		t.Fatal("expected unsupported record type error")
	}

	var dnsErr *DNSError
	if !errors.As(err, &dnsErr) {
		t.Fatalf("expected DNSError, got %T", err)
	}
	if dnsErr.Target != "example.com" {
		t.Fatalf("expected normalized target %q, got %q", "example.com", dnsErr.Target)
	}
	if !strings.Contains(dnsErr.Error(), "unsupported record type") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestLookupSupportedRecordTypes(t *testing.T) {
	original := resolverFactory
	t.Cleanup(func() { resolverFactory = original })

	tests := []struct {
		name       string
		recordType string
		resolver   dnsResolver
		check      func(t *testing.T, result *DNSResult)
	}{
		{
			name:       "A",
			recordType: RecordTypeA,
			resolver: fakeDNSResolver{
				lookupIPFunc: func(ctx context.Context, network, host string) ([]net.IP, error) {
					if network != "ip4" || host != "example.com" {
						t.Fatalf("unexpected A lookup inputs: network=%q host=%q", network, host)
					}
					return []net.IP{net.ParseIP("192.0.2.10")}, nil
				},
			},
			check: func(t *testing.T, result *DNSResult) {
				if len(result.Records) != 1 || result.Records[0].Type != RecordTypeA || result.Records[0].Value != "192.0.2.10" {
					t.Fatalf("unexpected A records: %+v", result.Records)
				}
			},
		},
		{
			name:       "AAAA",
			recordType: RecordTypeAAAA,
			resolver: fakeDNSResolver{
				lookupIPFunc: func(ctx context.Context, network, host string) ([]net.IP, error) {
					if network != "ip6" || host != "example.com" {
						t.Fatalf("unexpected AAAA lookup inputs: network=%q host=%q", network, host)
					}
					return []net.IP{net.ParseIP("2001:db8::10")}, nil
				},
			},
			check: func(t *testing.T, result *DNSResult) {
				if len(result.Records) != 1 || result.Records[0].Type != RecordTypeAAAA || result.Records[0].Value != "2001:db8::10" {
					t.Fatalf("unexpected AAAA records: %+v", result.Records)
				}
			},
		},
		{
			name:       "MX",
			recordType: RecordTypeMX,
			resolver: fakeDNSResolver{
				lookupMXFunc: func(ctx context.Context, name string) ([]*net.MX, error) {
					return []*net.MX{{Host: "mail.example.com.", Pref: 10}}, nil
				},
			},
			check: func(t *testing.T, result *DNSResult) {
				if len(result.Records) != 1 || result.Records[0].Type != RecordTypeMX || result.Records[0].Priority != 10 || result.Records[0].Value != "mail.example.com" {
					t.Fatalf("unexpected MX records: %+v", result.Records)
				}
			},
		},
		{
			name:       "TXT",
			recordType: RecordTypeTXT,
			resolver: fakeDNSResolver{
				lookupTXTFunc: func(ctx context.Context, name string) ([]string, error) {
					return []string{"v=spf1 include:example.com ~all"}, nil
				},
			},
			check: func(t *testing.T, result *DNSResult) {
				if len(result.Records) != 1 || result.Records[0].Type != RecordTypeTXT {
					t.Fatalf("unexpected TXT records: %+v", result.Records)
				}
			},
		},
		{
			name:       "CNAME",
			recordType: RecordTypeCNAME,
			resolver: fakeDNSResolver{
				lookupCNAMEFunc: func(ctx context.Context, host string) (string, error) {
					return "alias.example.com.", nil
				},
			},
			check: func(t *testing.T, result *DNSResult) {
				if len(result.Records) != 1 || result.Records[0].Type != RecordTypeCNAME || result.Records[0].Value != "alias.example.com" {
					t.Fatalf("unexpected CNAME records: %+v", result.Records)
				}
			},
		},
		{
			name:       "NS",
			recordType: RecordTypeNS,
			resolver: fakeDNSResolver{
				lookupNSFunc: func(ctx context.Context, name string) ([]*net.NS, error) {
					return []*net.NS{{Host: "ns1.example.com."}}, nil
				},
			},
			check: func(t *testing.T, result *DNSResult) {
				if len(result.Records) != 1 || result.Records[0].Type != RecordTypeNS || result.Records[0].Value != "ns1.example.com" {
					t.Fatalf("unexpected NS records: %+v", result.Records)
				}
			},
		},
		{
			name:       "ALL",
			recordType: RecordTypeALL,
			resolver: fakeDNSResolver{
				lookupIPFunc: func(ctx context.Context, network, host string) ([]net.IP, error) {
					switch network {
					case "ip4":
						return []net.IP{net.ParseIP("192.0.2.10")}, nil
					case "ip6":
						return []net.IP{net.ParseIP("2001:db8::10")}, nil
					default:
						return nil, nil
					}
				},
				lookupMXFunc: func(ctx context.Context, name string) ([]*net.MX, error) {
					return []*net.MX{{Host: "mail.example.com.", Pref: 5}}, nil
				},
				lookupTXTFunc: func(ctx context.Context, name string) ([]string, error) {
					return []string{"txt-record"}, nil
				},
				lookupCNAMEFunc: func(ctx context.Context, host string) (string, error) {
					return "alias.example.com.", nil
				},
				lookupNSFunc: func(ctx context.Context, name string) ([]*net.NS, error) {
					return []*net.NS{{Host: "ns1.example.com."}}, nil
				},
			},
			check: func(t *testing.T, result *DNSResult) {
				if len(result.Records) != 6 {
					t.Fatalf("expected 6 records from ALL lookup, got %+v", result.Records)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolverFactory = func(opts LookupOptions) dnsResolver {
				return tt.resolver
			}

			result, err := Lookup(" example.com. ", LookupOptions{
				RecordType: tt.recordType,
				Server:     "8.8.8.8",
				Timeout:    time.Second,
			})
			if err != nil {
				t.Fatalf("Lookup returned error: %v", err)
			}
			if result.Domain != "example.com" || result.QueryType != tt.recordType || result.Server != "8.8.8.8" {
				t.Fatalf("unexpected lookup metadata: %+v", result)
			}
			if result.QueryTime <= 0 {
				t.Fatalf("expected query time to be recorded, got %v", result.QueryTime)
			}
			tt.check(t, result)
		})
	}
}

func TestLookupWrapsResolverErrors(t *testing.T) {
	original := resolverFactory
	t.Cleanup(func() { resolverFactory = original })

	root := errors.New("lookup failed")
	resolverFactory = func(opts LookupOptions) dnsResolver {
		return fakeDNSResolver{
			lookupIPFunc: func(ctx context.Context, network, host string) ([]net.IP, error) {
				return nil, root
			},
		}
	}

	_, err := Lookup("example.com", LookupOptions{
		RecordType: RecordTypeA,
		Timeout:    time.Second,
	})
	if err == nil {
		t.Fatal("expected resolver error")
	}
	if !errors.Is(err, root) {
		t.Fatalf("expected wrapped resolver error, got %v", err)
	}
	var dnsErr *DNSError
	if !errors.As(err, &dnsErr) {
		t.Fatalf("expected DNSError, got %T", err)
	}
	if dnsErr.Operation != "lookup" || dnsErr.Target != "example.com" {
		t.Fatalf("unexpected DNS error context: %+v", dnsErr)
	}
}

func TestReverseLookupValidation(t *testing.T) {
	_, err := ReverseLookup("", time.Second)
	if err == nil {
		t.Fatal("expected empty IP error")
	}
	if !errors.Is(err, ErrEmptyIP) {
		t.Fatalf("expected ErrEmptyIP, got %v", err)
	}

	_, err = ReverseLookup("not-an-ip", time.Second)
	if err == nil {
		t.Fatal("expected invalid IP error")
	}
	if !errors.Is(err, ErrInvalidIP) {
		t.Fatalf("expected ErrInvalidIP, got %v", err)
	}
}

func TestReverseLookupUsesResolver(t *testing.T) {
	original := reverseLookupResolver
	t.Cleanup(func() { reverseLookupResolver = original })

	t.Run("success", func(t *testing.T) {
		reverseLookupResolver = fakeReverseResolver{
			lookupAddrFunc: func(ctx context.Context, addr string) ([]string, error) {
				if addr != "192.0.2.10" {
					t.Fatalf("unexpected reverse lookup address: %q", addr)
				}
				return []string{"host1.example.com.", "host2.example.com."}, nil
			},
		}

		result, err := ReverseLookup("192.0.2.10", time.Second)
		if err != nil {
			t.Fatalf("ReverseLookup returned error: %v", err)
		}
		if len(result.Hostnames) != 2 || result.Hostnames[0] != "host1.example.com" || result.Hostnames[1] != "host2.example.com" {
			t.Fatalf("unexpected reverse lookup hostnames: %+v", result.Hostnames)
		}
		if result.QueryTime <= 0 {
			t.Fatalf("expected query time to be recorded, got %v", result.QueryTime)
		}
	})

	t.Run("resolver error", func(t *testing.T) {
		root := errors.New("reverse lookup failed")
		reverseLookupResolver = fakeReverseResolver{
			lookupAddrFunc: func(ctx context.Context, addr string) ([]string, error) {
				return nil, root
			},
		}

		_, err := ReverseLookup("192.0.2.10", time.Second)
		if err == nil {
			t.Fatal("expected reverse lookup error")
		}
		if !errors.Is(err, root) {
			t.Fatalf("expected wrapped resolver error, got %v", err)
		}
	})
}

func TestCreateResolver(t *testing.T) {
	t.Run("default resolver", func(t *testing.T) {
		resolver := createResolver(LookupOptions{})
		if resolver != net.DefaultResolver {
			t.Fatal("expected default resolver")
		}
	})

	t.Run("custom server appends default port", func(t *testing.T) {
		original := resolverDialContext
		t.Cleanup(func() { resolverDialContext = original })

		var gotNetwork, gotAddress string
		var gotTimeout time.Duration
		resolverDialContext = func(ctx context.Context, network, address string, timeout time.Duration) (net.Conn, error) {
			gotNetwork = network
			gotAddress = address
			gotTimeout = timeout
			left, right := net.Pipe()
			_ = right.Close()
			return left, nil
		}

		resolver, ok := createResolver(LookupOptions{
			Server:  "127.0.0.1",
			Timeout: time.Second,
		}).(*net.Resolver)
		if !ok {
			t.Fatal("expected *net.Resolver for custom server")
		}

		conn, err := resolver.Dial(context.Background(), "udp", "ignored:53")
		if err != nil {
			t.Fatalf("resolver dial failed: %v", err)
		}
		defer conn.Close()
		if gotNetwork != "udp" || gotAddress != "127.0.0.1:53" || gotTimeout != time.Second {
			t.Fatalf("unexpected dial inputs: network=%q address=%q timeout=%v", gotNetwork, gotAddress, gotTimeout)
		}
	})

	t.Run("custom server uses explicit port", func(t *testing.T) {
		original := resolverDialContext
		t.Cleanup(func() { resolverDialContext = original })

		var gotNetwork, gotAddress string
		var gotTimeout time.Duration
		resolverDialContext = func(ctx context.Context, network, address string, timeout time.Duration) (net.Conn, error) {
			gotNetwork = network
			gotAddress = address
			gotTimeout = timeout
			left, right := net.Pipe()
			_ = right.Close()
			return left, nil
		}

		const serverAddress = "127.0.0.1:5353"
		resolver, ok := createResolver(LookupOptions{
			Server:  serverAddress,
			Timeout: time.Second,
		}).(*net.Resolver)
		if !ok {
			t.Fatal("expected *net.Resolver for custom server")
		}

		conn, err := resolver.Dial(context.Background(), "udp", "ignored:53")
		if err != nil {
			t.Fatalf("resolver dial failed: %v", err)
		}
		defer conn.Close()
		if gotNetwork != "udp" || gotAddress != serverAddress || gotTimeout != time.Second {
			t.Fatalf("unexpected dial inputs: network=%q address=%q timeout=%v", gotNetwork, gotAddress, gotTimeout)
		}
	})
}

func TestDNSError(t *testing.T) {
	root := errors.New("lookup failed")
	err := NewDNSError("lookup", "example.com", root)

	if !strings.Contains(err.Error(), `dns lookup error for "example.com": lookup failed`) {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	if !errors.Is(err, root) {
		t.Fatalf("expected wrapped error to match root cause")
	}
}
