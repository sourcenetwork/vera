#!/usr/bin/bash
# dial-peers.sh informs node1 of the addresses of its peers.
# This script performs a tendermint / cometbft rpc call to the remaining validators and queries their Id.
# The ID is used to construct the peer address of the persistent peers in node1.
#
# Note: this script makes use of an 'unsafe' Tendermint RPC call, which is disabled by default.
# To enable unsafe calls, the setting "unsafe" under p2p should be set to true in configs/config.toml

set -e

sleep 2

NODE2_ID="$(curl 'http://node2:26657/status' 2>/dev/null | jq --raw-output '.result.node_info.id')"
echo "Node 2: $NODE2_ID"

NODE3_ID="$(curl 'http://node3:26657/status' 2>/dev/null | jq --raw-output '.result.node_info.id')"
echo "Node 3: $NODE3_ID"

PEERS='\["'$NODE2_ID'@node2:26656","'$NODE3_ID'@node3:26656''"\]'
echo "Peers: $PEERS"

curl -i "http://node1:26657/dial_peers?persistent=true&peers=$PEERS" 2>/dev/null
