#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
ROOT_DIR=$(cd -- "$SCRIPT_DIR/../.." && pwd)
BIN_PATH=${1:-"$ROOT_DIR/bin/cidrator"}

EXPECTED_PMTU=${EXPECTED_PMTU:-1400}
CLIENT_LINK_MTU=${CLIENT_LINK_MTU:-1500}
MAX_PROBE_SIZE=${MAX_PROBE_SIZE:-1600}
MAX_HOPS=${MAX_HOPS:-4}

CLIENT_NS="cidrator-hop-client-$$"
ROUTER_NS="cidrator-hop-router-$$"
PEER_NS="cidrator-hop-peer-$$"

CLIENT_IF="hop-cl0"
ROUTER_CLIENT_IF="hop-rtcl0"
PEER_IF="hop-pr0"
ROUTER_PEER_IF="hop-rtpr0"

CLIENT_IP="10.50.0.2/24"
CLIENT_ADDR="${CLIENT_IP%/*}"
ROUTER_CLIENT_IP="10.50.0.1/24"
ROUTER_CLIENT_ADDR="${ROUTER_CLIENT_IP%/*}"
PEER_IP="10.60.0.2/24"
PEER_ADDR="${PEER_IP%/*}"
ROUTER_PEER_IP="10.60.0.1/24"
ROUTER_PEER_ADDR="${ROUTER_PEER_IP%/*}"

WORK_DIR=$(mktemp -d)
HOPS_RESULT="$WORK_DIR/hops.json"

require_command() {
	if ! command -v "$1" >/dev/null 2>&1; then
		echo "missing required command: $1" >&2
		exit 1
	fi
}

cleanup() {
	sudo ip netns del "$CLIENT_NS" >/dev/null 2>&1 || true
	sudo ip netns del "$ROUTER_NS" >/dev/null 2>&1 || true
	sudo ip netns del "$PEER_NS" >/dev/null 2>&1 || true
	rm -rf "$WORK_DIR"
}

print_debug() {
	echo "Hop-by-hop namespace lab failed. Current discovery output:" >&2
	if [[ -f "$HOPS_RESULT" ]]; then
		cat "$HOPS_RESULT" >&2 || true
	fi
}

run_discovery() {
	echo "Running hop-by-hop ICMP discovery inside $CLIENT_NS..."
	sudo ip netns exec "$CLIENT_NS" "$BIN_PATH" mtu discover "$PEER_ADDR" \
		--proto icmp \
		--hops \
		--max "$MAX_PROBE_SIZE" \
		--max-hops "$MAX_HOPS" \
		--pps 2 \
		--json \
		--quiet >"$HOPS_RESULT"

	python3 - "$HOPS_RESULT" "$PEER_ADDR" "$MAX_PROBE_SIZE" "$EXPECTED_PMTU" "$ROUTER_CLIENT_ADDR" "$CLIENT_LINK_MTU" <<'PY'
import json
import sys

path, expected_target, expected_max_probe_size, expected_pmtu, expected_router, expected_first_hop_mtu = sys.argv[1:]
expected_max_probe_size = int(expected_max_probe_size)
expected_pmtu = int(expected_pmtu)
expected_first_hop_mtu = int(expected_first_hop_mtu)

with open(path, "r", encoding="utf-8") as handle:
    data = json.load(handle)

errors = []
if data.get("target") != expected_target:
    errors.append(f'target={data.get("target")} expected {expected_target}')
if data.get("protocol") != "icmp":
    errors.append(f'protocol={data.get("protocol")} expected icmp')
if data.get("max_probe_size") != expected_max_probe_size:
    errors.append(f'max_probe_size={data.get("max_probe_size")} expected {expected_max_probe_size}')
if data.get("final_pmtu") != expected_pmtu:
    errors.append(f'final_pmtu={data.get("final_pmtu")} expected {expected_pmtu}')

hops = data.get("hops")
if not isinstance(hops, list) or len(hops) != 2:
    errors.append(f'hops length={len(hops) if isinstance(hops, list) else "invalid"} expected 2')
else:
    first, last = hops
    if first.get("hop") != 1:
        errors.append(f'first hop index={first.get("hop")} expected 1')
    if first.get("addr") != expected_router:
        errors.append(f'first hop addr={first.get("addr")} expected {expected_router}')
    if first.get("mtu") != expected_first_hop_mtu:
        errors.append(f'first hop mtu={first.get("mtu")} expected {expected_first_hop_mtu}')
    if not isinstance(first.get("rtt"), (int, float)) or first.get("rtt") < 0:
        errors.append(f'first hop rtt={first.get("rtt")} expected non-negative number')

    if last.get("hop") != 2:
        errors.append(f'last hop index={last.get("hop")} expected 2')
    if last.get("addr") != expected_target:
        errors.append(f'last hop addr={last.get("addr")} expected {expected_target}')
    if last.get("mtu") != expected_pmtu:
        errors.append(f'last hop mtu={last.get("mtu")} expected {expected_pmtu}')
    if not isinstance(last.get("rtt"), (int, float)) or last.get("rtt") < 0:
        errors.append(f'last hop rtt={last.get("rtt")} expected non-negative number')

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
require_command ping
require_command python3
require_command sysctl

if ! sudo -n true >/dev/null 2>&1; then
	echo "this lab requires passwordless sudo" >&2
	exit 1
fi

if [[ ! -x "$BIN_PATH" ]]; then
	echo "binary not found or not executable: $BIN_PATH" >&2
	exit 1
fi

if (( CLIENT_LINK_MTU < EXPECTED_PMTU )); then
	echo "CLIENT_LINK_MTU must be greater than or equal to EXPECTED_PMTU" >&2
	exit 1
fi

echo "Creating hop-by-hop MTU namespace lab..."
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

sudo ip -n "$CLIENT_NS" link set "$CLIENT_IF" mtu "$CLIENT_LINK_MTU" up
sudo ip -n "$ROUTER_NS" link set "$ROUTER_CLIENT_IF" mtu "$CLIENT_LINK_MTU" up
sudo ip -n "$PEER_NS" link set "$PEER_IF" mtu 1500 up
sudo ip -n "$ROUTER_NS" link set "$ROUTER_PEER_IF" mtu "$EXPECTED_PMTU" up

sudo ip -n "$CLIENT_NS" route add "$PEER_ADDR/32" via "$ROUTER_CLIENT_ADDR"
sudo ip -n "$PEER_NS" route add "$CLIENT_ADDR/32" via "$ROUTER_PEER_ADDR"
sudo ip netns exec "$ROUTER_NS" sysctl -w net.ipv4.ip_forward=1 >/dev/null
# Hop-by-hop ICMP probing is response-heavy; disable router-side ICMP throttling
# so exact-boundary probes are not misclassified as timeouts in CI.
sudo ip netns exec "$ROUTER_NS" sysctl -w net.ipv4.icmp_ratelimit=0 >/dev/null

echo "Checking baseline connectivity..."
sudo ip netns exec "$CLIENT_NS" ping -c 1 -W 1 "$PEER_ADDR" >/dev/null

run_discovery

echo "Hop-by-hop MTU lab passed."
