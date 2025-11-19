#!/bin/bash
set -e

rm -rf ~/.sourcehub-full || true

BIN="build/sourcehubd"
CHAIN_ID="sourcehub-dev"
HOME_DIR="$HOME/.sourcehub-full"
SNAPSHOT_INTERVAL=50
P2P=tcp://0.0.0.0:27684
ADDR=tcp://0.0.0.0:27685
RPC=tcp://127.0.0.1:27686
GRPC=localhost:9095
PPROF=localhost:6065

sedi() {
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "$@"
  else
    sed -i "$@"
  fi
}

echo "Initializing full node..."
$BIN init full-node --chain-id $CHAIN_ID --home $HOME_DIR

rsync -a --exclude priv_validator_state.json ~/.sourcehub/data/ $HOME_DIR/data/

sedi 's/^minimum-gas-prices = .*/minimum-gas-prices = "0.001uopen,0.001ucredit"/' $HOME_DIR/config/app.toml
sedi 's/^enable = .*/enable = true/' $HOME_DIR/config/app.toml

# Enable storing snapshots
sedi "s/^snapshot-interval = .*/snapshot-interval = $SNAPSHOT_INTERVAL/" $HOME_DIR/config/app.toml

NODE_ID=$($BIN tendermint show-node-id --home ~/.sourcehub)
# NODE_ID=$(curl -s http://localhost:26657/status | jq -r '.result.node_info.id')
sedi "s|^#* *persistent_peers *=.*|persistent_peers = \"$NODE_ID@0.0.0.0:26656\"|" $HOME_DIR/config/config.toml

# For local setups
sedi "s/^allow_duplicate_ip *=.*/allow_duplicate_ip = true/" $HOME_DIR/config/config.toml
sedi "s/^addr_book_strict *=.*/addr_book_strict = false/" $HOME_DIR/config/config.toml

echo "Starting full node with snapshot serving..."
$BIN start \
  --home $HOME_DIR \
  --p2p.laddr $P2P \
  --address $ADDR \
  --rpc.laddr $RPC \
  --grpc.address $GRPC \
  --rpc.pprof_laddr $PPROF