#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ROOT_DIR=$(cd -- "$SCRIPT_DIR/../.." && pwd)
BIN_PATH=${1:-"$ROOT_DIR/bin/cidrator"}

EXPECTED_PMTU=${EXPECTED_PMTU:-1400}
PEER_LINK_MTU=${PEER_LINK_MTU:-1500}
PEER_PORT=${PEER_PORT:-4821}

CLIENT_NS="cidrator-mtu-client-$$"
ROUTER_NS="cidrator-mtu-router-$$"
PEER_NS="cidrator-mtu-peer-$$"

CLIENT_IF="mtu-cl0"
ROUTER_CLIENT_IF="mtu-rtcl0"
PEER_IF="mtu-pr0"
ROUTER_PEER_IF="mtu-rtpr0"

CLIENT_IP="10.10.0.2/24"
CLIENT_ADDR="${CLIENT_IP%/*}"
ROUTER_CLIENT_IP="10.10.0.1/24"
ROUTER_CLIENT_ADDR="${ROUTER_CLIENT_IP%/*}"
PEER_IP="10.20.0.2/24"
PEER_ADDR="${PEER_IP%/*}"
ROUTER_PEER_IP="10.20.0.1/24"
ROUTER_PEER_ADDR="${ROUTER_PEER_IP%/*}"

WORK_DIR=$(mktemp -d)
PEER_LOG="$WORK_DIR/peer.log"
UDP_RESULT="$WORK_DIR/udp.json"
TCP_RESULT="$WORK_DIR/tcp.json"
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
	echo "Namespace lab failed. Current peer log:" >&2
	if [[ -f "$PEER_LOG" ]]; then
		cat "$PEER_LOG" >&2 || true
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

run_discovery() {
	local proto=$1
	local output=$2

	echo "Running $proto MTU discovery inside $CLIENT_NS..."
	sudo ip netns exec "$CLIENT_NS" "$BIN_PATH" mtu discover "$PEER_ADDR" \
		--proto "$proto" \
		--port "$PEER_PORT" \
		--min 576 \
		--max 1600 \
		--json \
		--quiet >"$output"

	python3 - "$output" "$proto" "$PEER_ADDR" "$EXPECTED_PMTU" <<'PY'
import json
import sys

path, expected_proto, expected_target, expected_pmtu = sys.argv[1:]
expected_pmtu = int(expected_pmtu)

with open(path, "r", encoding="utf-8") as handle:
    data = json.load(handle)

errors = []
if data.get("target") != expected_target:
    errors.append(f'target={data.get("target")} expected {expected_target}')
if data.get("protocol") != expected_proto:
    errors.append(f'protocol={data.get("protocol")} expected {expected_proto}')
if data.get("pmtu") != expected_pmtu:
    errors.append(f'pmtu={data.get("pmtu")} expected {expected_pmtu}')
expected_mss = expected_pmtu - 40
if data.get("mss") != expected_mss:
    errors.append(f'mss={data.get("mss")} expected {expected_mss}')
if data.get("hops", 0) < 1:
    errors.append(f'hops={data.get("hops")} expected >= 1')

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

echo "Creating MTU namespace lab..."
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
	--proto udp,tcp \
	--port "$PEER_PORT" \
	--response-pps 0 >"$PEER_LOG" 2>&1 &
PEER_PID=$!

wait_for_peer

echo "Checking basic connectivity..."
sudo ip netns exec "$CLIENT_NS" ping -c 1 -W 1 "$PEER_ADDR" >/dev/null

run_discovery udp "$UDP_RESULT"
run_discovery tcp "$TCP_RESULT"

echo "Namespace MTU lab passed."
