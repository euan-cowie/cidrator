package mtu

import (
	"context"
	"net"
)

// NetworkProber defines the interface for MTU probing operations
type NetworkProber interface {
	Probe(ctx context.Context, size int) *ProbeResult
	Close() error
}

// MTUDiscoveryInterface defines the interface for MTU discovery operations
type MTUDiscoveryInterface interface {
	DiscoverPMTU(ctx context.Context, minMTU, maxMTU int) (*MTUResult, error)
	Close() error
}

// NetworkResolver defines the interface for address resolution
type NetworkResolver interface {
	ResolveTarget(target string, ipv6 bool) (net.Addr, error)
}

// InterfaceDetector defines the interface for network interface detection
type InterfaceDetector interface {
	GetNetworkInterfaces() (*InterfaceResult, error)
	GetMaxMTU() (int, error)
}

// PacketSender defines the interface for sending network packets
type PacketSender interface {
	SendPacket(ctx context.Context, packet []byte, addr net.Addr) error
	ReceivePacket(ctx context.Context) ([]byte, net.Addr, error)
}
