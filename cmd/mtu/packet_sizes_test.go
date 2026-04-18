package mtu

import "testing"

func TestTCPMSSForMTU(t *testing.T) {
	tests := []struct {
		name string
		mtu  int
		ipv6 bool
		want int
	}{
		{name: "IPv4", mtu: 1500, ipv6: false, want: 1460},
		{name: "IPv6", mtu: 1500, ipv6: true, want: 1440},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tcpMSSForMTU(tt.mtu, tt.ipv6); got != tt.want {
				t.Fatalf("tcpMSSForMTU(%d, ipv6=%t) = %d, want %d", tt.mtu, tt.ipv6, got, tt.want)
			}
		})
	}
}

func TestPayloadSizeForPacket(t *testing.T) {
	tests := []struct {
		name     string
		packet   int
		overhead int
		want     int
	}{
		{name: "IPv4 UDP", packet: 1500, overhead: udpPacketOverhead(false), want: 1472},
		{name: "IPv6 UDP", packet: 1500, overhead: udpPacketOverhead(true), want: 1452},
		{name: "IPv4 TCP", packet: 1500, overhead: tcpPacketOverhead(false), want: 1460},
		{name: "negative clamps to zero", packet: 20, overhead: udpPacketOverhead(true), want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := payloadSizeForPacket(tt.packet, tt.overhead); got != tt.want {
				t.Fatalf("payloadSizeForPacket(%d, %d) = %d, want %d", tt.packet, tt.overhead, got, tt.want)
			}
		})
	}
}
