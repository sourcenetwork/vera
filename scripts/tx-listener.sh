#!/usr/bin/env sh

echo '{ "jsonrpc": "2.0","method": "subscribe","id": 0,"params": {"query": "tm.event='"'Tx'"'"} }' | websocat -n -t ws://127.0.0.1:26657/websocket | jq
