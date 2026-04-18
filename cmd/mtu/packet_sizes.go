package mtu

func tcpPacketOverhead(ipv6 bool) int {
	if ipv6 {
		return 60
	}
	return 40
}

func udpPacketOverhead(ipv6 bool) int {
	if ipv6 {
		return 48
	}
	return 28
}

func payloadSizeForPacket(packetSize, overhead int) int {
	payloadSize := packetSize - overhead
	if payloadSize < 0 {
		return 0
	}
	return payloadSize
}

func tcpMSSForMTU(mtu int, ipv6 bool) int {
	return mtu - tcpPacketOverhead(ipv6)
}
