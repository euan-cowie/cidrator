package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Record types supported by the lookup command
const (
	RecordTypeA     = "A"
	RecordTypeAAAA  = "AAAA"
	RecordTypeMX    = "MX"
	RecordTypeTXT   = "TXT"
	RecordTypeCNAME = "CNAME"
	RecordTypeNS    = "NS"
	RecordTypeALL   = "ALL"
)

// LookupOptions configures DNS lookup behavior
type LookupOptions struct {
	RecordType string        // Type of record to query (A, AAAA, MX, TXT, CNAME, NS, ALL)
	Server     string        // Custom DNS server (empty = system resolver)
	Timeout    time.Duration // Query timeout
	PreferIPv6 bool          // Prefer IPv6 results when available
}

// DefaultLookupOptions returns sensible defaults for DNS lookups
func DefaultLookupOptions() LookupOptions {
	return LookupOptions{
		RecordType: RecordTypeA,
		Timeout:    5 * time.Second,
	}
}

// DNSResult holds the results of a DNS lookup
type DNSResult struct {
	Domain    string        `json:"-" yaml:"-"`
	QueryType string        `json:"-" yaml:"-"`
	Records   []DNSRecord   `json:"-" yaml:"-"`
	QueryTime time.Duration `json:"-" yaml:"-"`
	Server    string        `json:"-" yaml:"-"`
}

// DNSRecord represents a single DNS record
type DNSRecord struct {
	Type     string `json:"type" yaml:"type"`
	Value    string `json:"value" yaml:"value"`
	Priority int    `json:"priority,omitempty" yaml:"priority,omitempty"` // For MX records
}

// ReverseResult holds the results of a reverse DNS lookup
type ReverseResult struct {
	IP        string        `json:"-" yaml:"-"`
	Hostnames []string      `json:"-" yaml:"-"`
	QueryTime time.Duration `json:"-" yaml:"-"`
}

// dnsResultOutput is the serialization-friendly version of DNSResult
type dnsResultOutput struct {
	Domain      string      `json:"domain" yaml:"domain"`
	QueryType   string      `json:"query_type" yaml:"query_type"`
	Records     []DNSRecord `json:"records" yaml:"records"`
	QueryTimeMS int64       `json:"query_time_ms" yaml:"query_time_ms"`
	Server      string      `json:"server,omitempty" yaml:"server,omitempty"`
}

// reverseResultOutput is the serialization-friendly version of ReverseResult
type reverseResultOutput struct {
	IP          string   `json:"ip" yaml:"ip"`
	Hostnames   []string `json:"hostnames" yaml:"hostnames"`
	QueryTimeMS int64    `json:"query_time_ms" yaml:"query_time_ms"`
}

// ToJSON converts DNSResult to JSON string
func (r *DNSResult) ToJSON() (string, error) {
	output := dnsResultOutput{
		Domain:      r.Domain,
		QueryType:   r.QueryType,
		Records:     r.Records,
		QueryTimeMS: r.QueryTime.Milliseconds(),
		Server:      r.Server,
	}
	bytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ToYAML converts DNSResult to YAML string
func (r *DNSResult) ToYAML() (string, error) {
	output := dnsResultOutput{
		Domain:      r.Domain,
		QueryType:   r.QueryType,
		Records:     r.Records,
		QueryTimeMS: r.QueryTime.Milliseconds(),
		Server:      r.Server,
	}
	bytes, err := yaml.Marshal(output)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ToJSON converts ReverseResult to JSON string
func (r *ReverseResult) ToJSON() (string, error) {
	output := reverseResultOutput{
		IP:          r.IP,
		Hostnames:   r.Hostnames,
		QueryTimeMS: r.QueryTime.Milliseconds(),
	}
	bytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ToYAML converts ReverseResult to YAML string
func (r *ReverseResult) ToYAML() (string, error) {
	output := reverseResultOutput{
		IP:          r.IP,
		Hostnames:   r.Hostnames,
		QueryTimeMS: r.QueryTime.Milliseconds(),
	}
	bytes, err := yaml.Marshal(output)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Lookup performs a DNS lookup for the specified domain
func Lookup(domain string, opts LookupOptions) (*DNSResult, error) {
	if domain == "" {
		return nil, NewDNSError("lookup", domain, ErrEmptyDomain)
	}

	// Clean the domain
	domain = strings.TrimSpace(domain)
	domain = strings.TrimSuffix(domain, ".")

	// Create resolver
	resolver := createResolver(opts)

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	start := time.Now()

	result := &DNSResult{
		Domain:    domain,
		QueryType: opts.RecordType,
		Server:    opts.Server,
		Records:   []DNSRecord{},
	}

	var err error
	switch strings.ToUpper(opts.RecordType) {
	case RecordTypeA:
		err = lookupA(ctx, resolver, domain, result)
	case RecordTypeAAAA:
		err = lookupAAAA(ctx, resolver, domain, result)
	case RecordTypeMX:
		err = lookupMX(ctx, resolver, domain, result)
	case RecordTypeTXT:
		err = lookupTXT(ctx, resolver, domain, result)
	case RecordTypeCNAME:
		err = lookupCNAME(ctx, resolver, domain, result)
	case RecordTypeNS:
		err = lookupNS(ctx, resolver, domain, result)
	case RecordTypeALL:
		err = lookupAll(ctx, resolver, domain, result)
	default:
		return nil, NewDNSError("lookup", domain, fmt.Errorf("unsupported record type: %s", opts.RecordType))
	}

	result.QueryTime = time.Since(start)

	if err != nil {
		return nil, err
	}

	return result, nil
}

// ReverseLookup performs a PTR record lookup for an IP address
func ReverseLookup(ip string, timeout time.Duration) (*ReverseResult, error) {
	if ip == "" {
		return nil, NewDNSError("reverse", ip, ErrEmptyIP)
	}

	// Validate IP address
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, NewDNSError("reverse", ip, ErrInvalidIP)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()

	names, err := net.DefaultResolver.LookupAddr(ctx, ip)
	if err != nil {
		return nil, NewDNSError("reverse", ip, err)
	}

	// Clean hostnames (remove trailing dots)
	hostnames := make([]string, len(names))
	for i, name := range names {
		hostnames[i] = strings.TrimSuffix(name, ".")
	}

	return &ReverseResult{
		IP:        ip,
		Hostnames: hostnames,
		QueryTime: time.Since(start),
	}, nil
}

// createResolver creates a DNS resolver with the given options
func createResolver(opts LookupOptions) *net.Resolver {
	if opts.Server == "" {
		return net.DefaultResolver
	}

	// Custom DNS server
	server := opts.Server
	if !strings.Contains(server, ":") {
		server = server + ":53"
	}

	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{Timeout: opts.Timeout}
			return d.DialContext(ctx, "udp", server)
		},
	}
}

// lookupA performs an A record lookup
func lookupA(ctx context.Context, resolver *net.Resolver, domain string, result *DNSResult) error {
	ips, err := resolver.LookupIP(ctx, "ip4", domain)
	if err != nil {
		return NewDNSError("lookup", domain, err)
	}

	for _, ip := range ips {
		result.Records = append(result.Records, DNSRecord{
			Type:  RecordTypeA,
			Value: ip.String(),
		})
	}
	return nil
}

// lookupAAAA performs an AAAA record lookup
func lookupAAAA(ctx context.Context, resolver *net.Resolver, domain string, result *DNSResult) error {
	ips, err := resolver.LookupIP(ctx, "ip6", domain)
	if err != nil {
		return NewDNSError("lookup", domain, err)
	}

	for _, ip := range ips {
		result.Records = append(result.Records, DNSRecord{
			Type:  RecordTypeAAAA,
			Value: ip.String(),
		})
	}
	return nil
}

// lookupMX performs an MX record lookup
func lookupMX(ctx context.Context, resolver *net.Resolver, domain string, result *DNSResult) error {
	mxs, err := resolver.LookupMX(ctx, domain)
	if err != nil {
		return NewDNSError("lookup", domain, err)
	}

	for _, mx := range mxs {
		result.Records = append(result.Records, DNSRecord{
			Type:     RecordTypeMX,
			Value:    strings.TrimSuffix(mx.Host, "."),
			Priority: int(mx.Pref),
		})
	}
	return nil
}

// lookupTXT performs a TXT record lookup
func lookupTXT(ctx context.Context, resolver *net.Resolver, domain string, result *DNSResult) error {
	txts, err := resolver.LookupTXT(ctx, domain)
	if err != nil {
		return NewDNSError("lookup", domain, err)
	}

	for _, txt := range txts {
		result.Records = append(result.Records, DNSRecord{
			Type:  RecordTypeTXT,
			Value: txt,
		})
	}
	return nil
}

// lookupCNAME performs a CNAME record lookup
func lookupCNAME(ctx context.Context, resolver *net.Resolver, domain string, result *DNSResult) error {
	cname, err := resolver.LookupCNAME(ctx, domain)
	if err != nil {
		return NewDNSError("lookup", domain, err)
	}

	result.Records = append(result.Records, DNSRecord{
		Type:  RecordTypeCNAME,
		Value: strings.TrimSuffix(cname, "."),
	})
	return nil
}

// lookupNS performs an NS record lookup
func lookupNS(ctx context.Context, resolver *net.Resolver, domain string, result *DNSResult) error {
	nss, err := resolver.LookupNS(ctx, domain)
	if err != nil {
		return NewDNSError("lookup", domain, err)
	}

	for _, ns := range nss {
		result.Records = append(result.Records, DNSRecord{
			Type:  RecordTypeNS,
			Value: strings.TrimSuffix(ns.Host, "."),
		})
	}
	return nil
}

// lookupAll performs all supported record lookups
func lookupAll(ctx context.Context, resolver *net.Resolver, domain string, result *DNSResult) error {
	// A records
	if ips, err := resolver.LookupIP(ctx, "ip4", domain); err == nil {
		for _, ip := range ips {
			result.Records = append(result.Records, DNSRecord{Type: RecordTypeA, Value: ip.String()})
		}
	}

	// AAAA records
	if ips, err := resolver.LookupIP(ctx, "ip6", domain); err == nil {
		for _, ip := range ips {
			result.Records = append(result.Records, DNSRecord{Type: RecordTypeAAAA, Value: ip.String()})
		}
	}

	// CNAME record
	if cname, err := resolver.LookupCNAME(ctx, domain); err == nil {
		result.Records = append(result.Records, DNSRecord{Type: RecordTypeCNAME, Value: strings.TrimSuffix(cname, ".")})
	}

	// MX records
	if mxs, err := resolver.LookupMX(ctx, domain); err == nil {
		for _, mx := range mxs {
			result.Records = append(result.Records, DNSRecord{
				Type:     RecordTypeMX,
				Value:    strings.TrimSuffix(mx.Host, "."),
				Priority: int(mx.Pref),
			})
		}
	}

	// NS records
	if nss, err := resolver.LookupNS(ctx, domain); err == nil {
		for _, ns := range nss {
			result.Records = append(result.Records, DNSRecord{Type: RecordTypeNS, Value: strings.TrimSuffix(ns.Host, ".")})
		}
	}

	// TXT records
	if txts, err := resolver.LookupTXT(ctx, domain); err == nil {
		for _, txt := range txts {
			result.Records = append(result.Records, DNSRecord{Type: RecordTypeTXT, Value: txt})
		}
	}

	return nil
}
