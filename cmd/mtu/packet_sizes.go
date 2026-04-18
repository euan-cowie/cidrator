package mtu

const tcpTimestampOptionBytes = 12

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

func tcpProbePayloadSize(packetSize, negotiatedMSS int, ipv6 bool) (int, bool) {
	targetPayload := payloadSizeForPacket(packetSize, tcpPacketOverhead(ipv6))
	if negotiatedMSS <= 0 {
		return targetPayload, true
	}

	if targetPayload-negotiatedMSS > tcpTimestampOptionBytes {
		return 0, false
	}

	if negotiatedMSS < targetPayload {
		return negotiatedMSS, true
	}

	return targetPayload, true
}
