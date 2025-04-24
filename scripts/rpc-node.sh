#!/bin/bash

set -e

rm -rf ~/.sourcehub-rpc || true

sedi() {
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "$@"
  else
    sed -i "$@"
  fi
}

BIN="build/sourcehubd"
CHAIN_ID="sourcehub-dev"
HOME_DIR="$HOME/.sourcehub-rpc"
RPC_SERVERS="http://localhost:27686,http://localhost:27686"
TRUST_HEIGHT=50
TRUST_HASH=$(curl -s http://localhost:27686/block?height=$TRUST_HEIGHT | jq -r '.result.block_id.hash')
P2P=tcp://0.0.0.0:27674
ADDR=tcp://0.0.0.0:27675
RPC=tcp://0.0.0.0:26676
GRPC=0.0.0.0:9094
PPROF=localhost:6064

echo "Initializing new node..."
$BIN init rpc-node --chain-id $CHAIN_ID --home $HOME_DIR
cp ~/.sourcehub/config/genesis.json $HOME_DIR/config/genesis.json

sedi 's/^minimum-gas-prices = .*/minimum-gas-prices = "0.001uopen,0.001ucredit"/' ~/.sourcehub/config/app.toml

# Enable API / GRPC
sedi 's/^enable = .*/enable = true/' ~/.sourcehub/config/app.toml

# Enable state sync from snapshots
sedi "s/^enable *=.*/enable = true/" $HOME_DIR/config/config.toml
sedi "s|^rpc_servers *=.*|rpc_servers = \"$RPC_SERVERS\"|" $HOME_DIR/config/config.toml
sedi "s|^trust_height *=.*|trust_height = $TRUST_HEIGHT|" $HOME_DIR/config/config.toml
sedi "s|^trust_hash *=.*|trust_hash = \"$TRUST_HASH\"|" $HOME_DIR/config/config.toml

# For local setups
sedi "s/^allow_duplicate_ip *=.*/allow_duplicate_ip = true/" $HOME_DIR/config/config.toml
sedi "s/^addr_book_strict *=.*/addr_book_strict = false/" $HOME_DIR/config/config.toml

NODE_ID=$(curl -s http://localhost:27686/status | jq -r '.result.node_info.id')
# NODE_ID=$($BIN tendermint show-node-id --home ~/.sourcehub-full)
sedi "s|^#* *persistent_peers *=.*|persistent_peers = \"$NODE_ID@0.0.0.0:27684\"|" $HOME_DIR/config/config.toml

echo "Starting new node (state sync)..."
$BIN start \
  --home $HOME_DIR \
  --p2p.laddr $P2P \
  --address $ADDR \
  --rpc.laddr $RPC \
  --grpc.address $GRPC \
  --rpc.pprof_laddr $PPROF