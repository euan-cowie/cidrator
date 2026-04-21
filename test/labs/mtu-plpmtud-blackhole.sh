#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ROOT_DIR=$(cd -- "$SCRIPT_DIR/../.." && pwd)
BIN_PATH=${1:-"$ROOT_DIR/bin/cidrator"}

EXPECTED_PMTU=${EXPECTED_PMTU:-1400}
PEER_LINK_MTU=${PEER_LINK_MTU:-1500}
PEER_PORT=${PEER_PORT:-4821}
PROBE_TIMEOUT=${PROBE_TIMEOUT:-500ms}

CLIENT_NS="cidrator-plp-client-$$"
ROUTER_NS="cidrator-plp-router-$$"
PEER_NS="cidrator-plp-peer-$$"

CLIENT_IF="plp-cl0"
ROUTER_CLIENT_IF="plp-rtcl0"
PEER_IF="plp-pr0"
ROUTER_PEER_IF="plp-rtpr0"

CLIENT_IP="10.30.0.2/24"
CLIENT_ADDR="${CLIENT_IP%/*}"
ROUTER_CLIENT_IP="10.30.0.1/24"
ROUTER_CLIENT_ADDR="${ROUTER_CLIENT_IP%/*}"
PEER_IP="10.40.0.2/24"
PEER_ADDR="${PEER_IP%/*}"
ROUTER_PEER_IP="10.40.0.1/24"
ROUTER_PEER_ADDR="${ROUTER_PEER_IP%/*}"

WORK_DIR=$(mktemp -d)
PEER_LOG="$WORK_DIR/peer.log"
ICMP_FAIL_LOG="$WORK_DIR/icmp-fail.log"
PLPMTUD_RESULT="$WORK_DIR/plpmtud.json"
PEER_PID=""

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "missing required command: $1" >&2
		exit 1
	fi
}

cleanup() {
	if [[ -n "$PEER_PID" ]]; then
		sudo kill "$PEER_PID" >/dev/null 2>&1 || true
		wait "$PEER_PID" >/dev/null 2>&1 || true
	fi

	sudo ip netns del "$CLIENT_NS" >/dev/null 2>&1 || true
	sudo ip netns del "$ROUTER_NS" >/dev/null 2>&1 || true
	sudo ip netns del "$PEER_NS" >/dev/null 2>&1 || true
	rm -rf "$WORK_DIR"
}

print_debug() {
	echo "PLPMTUD black-hole lab failed. Current peer log:" >&2
	if [[ -f "$PEER_LOG" ]]; then
		cat "$PEER_LOG" >&2 || true
	fi
	if [[ -f "$ICMP_FAIL_LOG" ]]; then
		echo "ICMP discovery output:" >&2
		cat "$ICMP_FAIL_LOG" >&2 || true
	fi
	if [[ -f "$PLPMTUD_RESULT" ]]; then
		echo "PLPMTUD discovery output:" >&2
		cat "$PLPMTUD_RESULT" >&2 || true
	fi
}

wait_for_peer() {
	local attempts=0
	while (( attempts < 50 )); do
		if sudo ip netns exec "$PEER_NS" ss -ltnu | grep -q ":$PEER_PORT\\b"; then
			return 0
		fi
		sleep 0.1
		attempts=$((attempts + 1))
	done

	echo "peer endpoint did not start listening on port $PEER_PORT" >&2
	return 1
}

apply_icmp_blackhole() {
	echo "Applying ICMP black-hole rules in $ROUTER_NS..."
	sudo ip netns exec "$ROUTER_NS" iptables -A FORWARD -p icmp -j DROP
	sudo ip netns exec "$ROUTER_NS" iptables -A OUTPUT -p icmp -d "$CLIENT_ADDR" -j DROP
}

expect_icmp_discovery_failure() {
	echo "Running plain ICMP discovery and expecting failure..."
	if sudo ip netns exec "$CLIENT_NS" "$BIN_PATH" mtu discover "$PEER_ADDR" \
		--proto icmp \
		--min 576 \
		--max 1600 \
		--timeout "$PROBE_TIMEOUT" \
		--quiet >"$ICMP_FAIL_LOG" 2>&1; then
		echo "plain ICMP discovery unexpectedly succeeded under ICMP black-hole conditions" >&2
		return 1
	fi

	if ! grep -Eq "no working MTU found|MTU discovery failed" "$ICMP_FAIL_LOG"; then
		echo "plain ICMP discovery failed for an unexpected reason" >&2
		return 1
	fi
}

run_plpmtud_discovery() {
	echo "Running ICMP discovery with PLPMTUD fallback..."
	sudo ip netns exec "$CLIENT_NS" "$BIN_PATH" mtu discover "$PEER_ADDR" \
		--proto icmp \
		--plpmtud \
		--plp-port "$PEER_PORT" \
		--min 576 \
		--max 1600 \
		--timeout "$PROBE_TIMEOUT" \
		--json \
		--quiet >"$PLPMTUD_RESULT"

	python3 - "$PLPMTUD_RESULT" "$PEER_ADDR" "$EXPECTED_PMTU" <<'PY'
import json
import sys

path, expected_target, expected_pmtu = sys.argv[1:]
expected_pmtu = int(expected_pmtu)

with open(path, "r", encoding="utf-8") as handle:
    data = json.load(handle)

errors = []
if data.get("target") != expected_target:
    errors.append(f'target={data.get("target")} expected {expected_target}')
if data.get("protocol") != "plpmtud":
    errors.append(f'protocol={data.get("protocol")} expected plpmtud')
if data.get("pmtu") != expected_pmtu:
    errors.append(f'pmtu={data.get("pmtu")} expected {expected_pmtu}')
expected_mss = expected_pmtu - 40
if data.get("mss") != expected_mss:
    errors.append(f'mss={data.get("mss")} expected {expected_mss}')
if data.get("hops") != 0:
    errors.append(f'hops={data.get("hops")} expected 0')

if errors:
    raise SystemExit("; ".join(errors))
PY
}

trap print_debug ERR
trap cleanup EXIT

if [[ "$(uname -s)" != "Linux" ]]; then
	echo "this lab requires Linux network namespaces" >&2
	exit 1
fi

require_command sudo
require_command ip
require_command ss
require_command ping
require_command python3
require_command iptables

if ! sudo -n true >/dev/null 2>&1; then
	echo "this lab requires passwordless sudo" >&2
	exit 1
fi

if [[ ! -x "$BIN_PATH" ]]; then
	echo "binary not found or not executable: $BIN_PATH" >&2
	exit 1
fi

if (( PEER_LINK_MTU < EXPECTED_PMTU )); then
	echo "PEER_LINK_MTU must be greater than or equal to EXPECTED_PMTU" >&2
	exit 1
fi

echo "Creating PLPMTUD black-hole lab..."
sudo ip netns add "$CLIENT_NS"
sudo ip netns add "$ROUTER_NS"
sudo ip netns add "$PEER_NS"

sudo ip link add "$CLIENT_IF" type veth peer name "$ROUTER_CLIENT_IF"
sudo ip link add "$PEER_IF" type veth peer name "$ROUTER_PEER_IF"

sudo ip link set "$CLIENT_IF" netns "$CLIENT_NS"
sudo ip link set "$ROUTER_CLIENT_IF" netns "$ROUTER_NS"
sudo ip link set "$PEER_IF" netns "$PEER_NS"
sudo ip link set "$ROUTER_PEER_IF" netns "$ROUTER_NS"

sudo ip -n "$CLIENT_NS" addr add "$CLIENT_IP" dev "$CLIENT_IF"
sudo ip -n "$ROUTER_NS" addr add "$ROUTER_CLIENT_IP" dev "$ROUTER_CLIENT_IF"
sudo ip -n "$PEER_NS" addr add "$PEER_IP" dev "$PEER_IF"
sudo ip -n "$ROUTER_NS" addr add "$ROUTER_PEER_IP" dev "$ROUTER_PEER_IF"

sudo ip -n "$CLIENT_NS" link set lo up
sudo ip -n "$ROUTER_NS" link set lo up
sudo ip -n "$PEER_NS" link set lo up

sudo ip -n "$CLIENT_NS" link set "$CLIENT_IF" mtu 1500 up
sudo ip -n "$ROUTER_NS" link set "$ROUTER_CLIENT_IF" mtu 1500 up
sudo ip -n "$PEER_NS" link set "$PEER_IF" mtu "$PEER_LINK_MTU" up
sudo ip -n "$ROUTER_NS" link set "$ROUTER_PEER_IF" mtu "$EXPECTED_PMTU" up

sudo ip -n "$CLIENT_NS" route add "$PEER_ADDR/32" via "$ROUTER_CLIENT_ADDR"
sudo ip -n "$PEER_NS" route add "$CLIENT_ADDR/32" via "$ROUTER_PEER_ADDR"
sudo ip netns exec "$ROUTER_NS" sysctl -w net.ipv4.ip_forward=1 >/dev/null

echo "Starting advanced peer endpoint in $PEER_NS..."
sudo ip netns exec "$PEER_NS" "$BIN_PATH" mtu peer \
	--listen "$PEER_ADDR" \
	--allow-remote \
	--proto udp \
	--port "$PEER_PORT" \
	--response-pps 0 >"$PEER_LOG" 2>&1 &
PEER_PID=$!

wait_for_peer

echo "Checking baseline connectivity before black-holing ICMP..."
sudo ip netns exec "$CLIENT_NS" ping -c 1 -W 1 "$PEER_ADDR" >/dev/null

apply_icmp_blackhole
expect_icmp_discovery_failure
run_plpmtud_discovery

echo "PLPMTUD black-hole lab passed."
