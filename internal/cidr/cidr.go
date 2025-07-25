package cidr

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net"
	"strings"

	"gopkg.in/yaml.v3"
)

// Constants for network calculations
const (
	IPv4Bits                = 32
	IPv6Bits                = 128
	MaxSafeExpansionSize    = 65536 // /16 for IPv4 equivalent
	DefaultSubnetOverhead   = 2     // Network + broadcast addresses
	MinPointToPointPrefixV4 = 31    // /31 networks for point-to-point
	HostRoutePrefixV4       = 32    // /32 host routes
	HostRoutePrefixV6       = 128   // /128 host routes
)

// ExpansionOptions holds configuration for IP address expansion
type ExpansionOptions struct {
	Limit int // Maximum number of IPs to expand (0 = no limit, subject to safety limits)
}

// DivisionOptions holds configuration for subnet division
type DivisionOptions struct {
	Parts int // Number of parts to divide the network into
}

// NetworkInfo represents detailed information about a CIDR network
type NetworkInfo struct {
	Network         *net.IPNet
	IP              net.IP
	BaseAddress     net.IP
	BroadcastAddr   net.IP
	FirstUsable     net.IP
	LastUsable      net.IP
	Netmask         net.IP
	HostMask        net.IP
	PrefixLength    int
	HostBits        int
	TotalAddresses  *big.Int
	UsableAddresses *big.Int
	IsIPv6          bool
}

// NetworkInfoOutput represents network info for structured output formats
type NetworkInfoOutput struct {
	BaseAddress     string `json:"base_address" yaml:"base_address"`
	BroadcastAddr   string `json:"broadcast_address,omitempty" yaml:"broadcast_address,omitempty"`
	FirstUsable     string `json:"first_usable" yaml:"first_usable"`
	LastUsable      string `json:"last_usable" yaml:"last_usable"`
	Netmask         string `json:"netmask" yaml:"netmask"`
	HostMask        string `json:"host_mask,omitempty" yaml:"host_mask,omitempty"`
	PrefixLength    int    `json:"prefix_length" yaml:"prefix_length"`
	HostBits        int    `json:"host_bits" yaml:"host_bits"`
	TotalAddresses  string `json:"total_addresses" yaml:"total_addresses"`
	UsableAddresses string `json:"usable_addresses" yaml:"usable_addresses"`
	IsIPv6          bool   `json:"is_ipv6" yaml:"is_ipv6"`
}

// ToOutput converts NetworkInfo to NetworkInfoOutput for structured formats
func (info *NetworkInfo) ToOutput() *NetworkInfoOutput {
	output := &NetworkInfoOutput{
		BaseAddress:     info.BaseAddress.String(),
		FirstUsable:     info.FirstUsable.String(),
		LastUsable:      info.LastUsable.String(),
		Netmask:         info.Netmask.String(),
		PrefixLength:    info.PrefixLength,
		HostBits:        info.HostBits,
		TotalAddresses:  FormatBigInt(info.TotalAddresses),
		UsableAddresses: FormatBigInt(info.UsableAddresses),
		IsIPv6:          info.IsIPv6,
	}

	if !info.IsIPv6 {
		if info.BroadcastAddr != nil {
			output.BroadcastAddr = info.BroadcastAddr.String()
		}
		if info.HostMask != nil {
			output.HostMask = info.HostMask.String()
		}
	}

	return output
}

// ToJSON converts NetworkInfo to JSON string
func (info *NetworkInfo) ToJSON() (string, error) {
	output := info.ToOutput()
	bytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ToYAML converts NetworkInfo to YAML string
func (info *NetworkInfo) ToYAML() (string, error) {
	output := info.ToOutput()
	bytes, err := yaml.Marshal(output)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// ParseCIDR parses a CIDR string and returns network information
func ParseCIDR(cidr string) (*NetworkInfo, error) {
	ip, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, NewCIDRError("parse", cidr, ErrInvalidCIDR)
	}

	info := &NetworkInfo{
		Network:      network,
		IP:           ip,
		BaseAddress:  network.IP,
		PrefixLength: getPrefixLength(network),
		IsIPv6:       ip.To4() == nil,
	}

	if info.IsIPv6 {
		return configureIPv6Network(info, network)
	}
	return configureIPv4Network(info, network)
}

// configureIPv6Network configures network information for IPv6 networks
func configureIPv6Network(info *NetworkInfo, network *net.IPNet) (*NetworkInfo, error) {
	info.HostBits = IPv6Bits - info.PrefixLength
	info.TotalAddresses = calculateTotalAddresses(info.HostBits)
	info.UsableAddresses = calculateUsableAddresses(info.TotalAddresses, info.HostBits)

	mask := net.CIDRMask(info.PrefixLength, IPv6Bits)
	info.Netmask = net.IP(mask)
	info.HostMask = getHostMask(info.Netmask)
	info.FirstUsable = info.BaseAddress
	info.LastUsable = getLastIPv6(network)

	return info, nil
}

// configureIPv4Network configures network information for IPv4 networks
func configureIPv4Network(info *NetworkInfo, network *net.IPNet) (*NetworkInfo, error) {
	info.HostBits = IPv4Bits - info.PrefixLength
	info.TotalAddresses = calculateTotalAddresses(info.HostBits)
	info.UsableAddresses = calculateUsableAddresses(info.TotalAddresses, info.HostBits)

	mask := net.CIDRMask(info.PrefixLength, IPv4Bits)
	info.Netmask = net.IP(mask)
	info.HostMask = getHostMask(info.Netmask)
	info.BroadcastAddr = getBroadcastAddress(network)
	info.FirstUsable = getFirstUsable(network)
	info.LastUsable = getLastUsable(network)

	return info, nil
}

// calculateTotalAddresses calculates total addresses for given host bits
func calculateTotalAddresses(hostBits int) *big.Int {
	return big.NewInt(0).Exp(big.NewInt(2), big.NewInt(int64(hostBits)), nil)
}

// calculateUsableAddresses calculates usable addresses (excluding network/broadcast if applicable)
func calculateUsableAddresses(totalAddresses *big.Int, hostBits int) *big.Int {
	usable := big.NewInt(0).Set(totalAddresses)
	if hostBits > 1 {
		usable.Sub(usable, big.NewInt(DefaultSubnetOverhead))
	}
	return usable
}

// Expand lists all IP addresses in a CIDR range
func Expand(cidr string, opts ExpansionOptions) ([]string, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, NewCIDRError("expand", cidr, ErrInvalidCIDR)
	}

	prefixLen, bits := network.Mask.Size()
	hostBits := bits - prefixLen

	// Calculate total addresses
	totalAddresses := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(int64(hostBits)), nil)

	// Check if the range is too large
	if opts.Limit > 0 && totalAddresses.Cmp(big.NewInt(int64(opts.Limit))) > 0 {
		return nil, NewCIDRError("expand", cidr, fmt.Errorf("range contains %s addresses, exceeds limit of %d", FormatBigInt(totalAddresses), opts.Limit))
	}

	// For very large ranges, we need to be careful about memory
	if totalAddresses.Cmp(big.NewInt(MaxSafeExpansionSize)) > 0 {
		return nil, NewCIDRError("expand", cidr, ErrTooLarge)
	}

	var ips []string
	currentIP := make(net.IP, len(network.IP))
	copy(currentIP, network.IP)

	// Convert total addresses to int for iteration
	totalInt := totalAddresses.Int64()

	for i := int64(0); i < totalInt; i++ {
		ips = append(ips, currentIP.String())

		// Increment IP address
		incrementIP(currentIP)
	}

	return ips, nil
}

// incrementIP increments an IP address by 1
func incrementIP(ip net.IP) {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
}

// Contains checks if an IP address is within the CIDR range
func Contains(cidr, ipStr string) (bool, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, NewCIDRError("contains", cidr, ErrInvalidCIDR)
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, NewValidationError("ip", ipStr, ErrInvalidIP)
	}

	return network.Contains(ip), nil
}

// Count returns the total number of addresses in a CIDR range
func Count(cidr string) (*big.Int, error) {
	info, err := ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}
	return info.TotalAddresses, nil
}

// Overlaps checks if two CIDR ranges overlap
func Overlaps(cidr1, cidr2 string) (bool, error) {
	_, net1, err := net.ParseCIDR(cidr1)
	if err != nil {
		return false, fmt.Errorf("invalid first CIDR: %v", err)
	}

	_, net2, err := net.ParseCIDR(cidr2)
	if err != nil {
		return false, fmt.Errorf("invalid second CIDR: %v", err)
	}

	// Check if either network contains the other's network address
	return net1.Contains(net2.IP) || net2.Contains(net1.IP), nil
}

// Divide splits a CIDR range into N smaller subnets
func Divide(cidr string, opts DivisionOptions) ([]string, error) {
	if opts.Parts <= 0 {
		return nil, NewValidationError("parts", fmt.Sprintf("%d", opts.Parts), ErrInvalidParts)
	}

	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, NewCIDRError("divide", cidr, ErrInvalidCIDR)
	}

	return generateSubnets(network, opts.Parts)
}

// generateSubnets creates the actual subnet list from a network and part count
func generateSubnets(network *net.IPNet, parts int) ([]string, error) {
	subnetConfig, err := calculateSubnetConfiguration(network, parts)
	if err != nil {
		return nil, err
	}

	return createSubnetList(network, subnetConfig, parts)
}

// SubnetConfiguration holds the calculated parameters for subnet division
type SubnetConfiguration struct {
	NewPrefixLen int
	Increment    *big.Int
	Bits         int
}

// calculateSubnetConfiguration determines the subnet parameters
func calculateSubnetConfiguration(network *net.IPNet, parts int) (*SubnetConfiguration, error) {
	prefixLen, bits := network.Mask.Size()
	isIPv6 := len(network.IP) == 16

	bitsNeeded := int(math.Ceil(math.Log2(float64(parts))))
	newPrefixLen := prefixLen + bitsNeeded

	if (isIPv6 && newPrefixLen > IPv6Bits) || (!isIPv6 && newPrefixLen > IPv4Bits) {
		return nil, ErrInsufficientBits
	}

	increment := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(int64(bits-newPrefixLen)), nil)

	return &SubnetConfiguration{
		NewPrefixLen: newPrefixLen,
		Increment:    increment,
		Bits:         bits,
	}, nil
}

// createSubnetList generates the list of subnet strings
func createSubnetList(network *net.IPNet, config *SubnetConfiguration, parts int) ([]string, error) {
	var subnets []string
	currentIP := make(net.IP, len(network.IP))
	copy(currentIP, network.IP)

	for i := 0; i < parts; i++ {
		subnet := &net.IPNet{
			IP:   make(net.IP, len(currentIP)),
			Mask: net.CIDRMask(config.NewPrefixLen, config.Bits),
		}
		copy(subnet.IP, currentIP)

		subnets = append(subnets, subnet.String())
		addToIP(currentIP, config.Increment)
	}

	return subnets, nil
}

// Helper functions

func getPrefixLength(network *net.IPNet) int {
	ones, _ := network.Mask.Size()
	return ones
}

func getHostMask(netmask net.IP) net.IP {
	hostMask := make(net.IP, len(netmask))
	for i := range netmask {
		hostMask[i] = ^netmask[i]
	}
	return hostMask
}

func getBroadcastAddress(network *net.IPNet) net.IP {
	broadcast := make(net.IP, len(network.IP))
	copy(broadcast, network.IP)

	for i := range broadcast {
		broadcast[i] |= ^network.Mask[i]
	}
	return broadcast
}

func getFirstUsable(network *net.IPNet) net.IP {
	prefixLen, _ := network.Mask.Size()
	if prefixLen >= MinPointToPointPrefixV4 {
		// For /31 and /32, first address is usable
		return network.IP
	}

	first := make(net.IP, len(network.IP))
	copy(first, network.IP)

	// Add 1 to the last byte
	for i := len(first) - 1; i >= 0; i-- {
		first[i]++
		if first[i] != 0 {
			break
		}
	}
	return first
}

func getLastUsable(network *net.IPNet) net.IP {
	prefixLen, _ := network.Mask.Size()
	if prefixLen >= MinPointToPointPrefixV4 {
		// For /31, both addresses are usable; for /32, only one
		broadcast := getBroadcastAddress(network)
		if prefixLen == HostRoutePrefixV4 {
			return broadcast
		}
		return broadcast
	}

	broadcast := getBroadcastAddress(network)
	last := make(net.IP, len(broadcast))
	copy(last, broadcast)

	// Subtract 1 from the last byte
	for i := len(last) - 1; i >= 0; i-- {
		if last[i] > 0 {
			last[i]--
			break
		}
		last[i] = 255
	}
	return last
}

func getLastIPv6(network *net.IPNet) net.IP {
	last := make(net.IP, len(network.IP))
	copy(last, network.IP)

	for i := range last {
		last[i] |= ^network.Mask[i]
	}
	return last
}

func addToIP(ip net.IP, increment *big.Int) {
	// Convert IP to big.Int, add increment, convert back
	ipInt := big.NewInt(0)
	ipInt.SetBytes(ip)
	ipInt.Add(ipInt, increment)

	bytes := ipInt.Bytes()

	// Pad with zeros if necessary
	if len(bytes) < len(ip) {
		padded := make([]byte, len(ip))
		copy(padded[len(ip)-len(bytes):], bytes)
		bytes = padded
	}

	copy(ip, bytes)
}

// FormatBigInt formats a big.Int with thousand separators
func FormatBigInt(n *big.Int) string {
	s := n.String()
	if len(s) <= 3 {
		return s
	}

	var result strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(r)
	}
	return result.String()
}
