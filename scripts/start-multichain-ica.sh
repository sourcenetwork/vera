#!/bin/sh

set -e

BIN="build/sourcehubd"

CHAIN1_ID="sourcehub-1"
CHAIN2_ID="sourcehub-2"

C1V1_HOME="$HOME/.sourcehub-1-1"
C2V1_HOME="$HOME/.sourcehub-2-1"
HERMES_HOME="$HOME/.hermes"

C1V1_GRPC=localhost:9091
C1V1_P2P=localhost:27665
C1V1_ADDR=tcp://0.0.0.0:27666
C1V1_RPC=tcp://127.0.0.1:26667
C1V1_PPROF=localhost:6061

C2V1_GRPC=localhost:9092
C2V1_P2P=localhost:27668
C2V1_ADDR=tcp://0.0.0.0:27669
C2V1_RPC=tcp://127.0.0.1:26670
C2V1_PPROF=localhost:6062

C2V1_PPROF=localhost:6062

C1V1_NAME="validator1-1-node"
C2V1_NAME="validator2-1-node"

C1V1="validator1-1"
C2V1="validator2-1"

ICA_PACKET_JSON="scripts/ica_packet.json"

POLICY_CONTENT="name: ica test policy"

# Allow only specific controller chain and connection ids to open ICA connections on the host
# export SOURCEHUB_ICA_ALLOWED_CONTROLLERS="sourcehub-1"
# export SOURCEHUB_ICA_ALLOWED_CONNECTIONS="connection-0"

# Exit if no hermes binary found
if ! type "hermes" > /dev/null; then
  echo "Hermes binary not found"
  exit 0
fi

# Kill running processes
killall sourcehubd 2>/dev/null || true
killall hermes 2>/dev/null || true

# Cleanup directories
rm -rf $C1V1_HOME
rm -rf $C2V1_HOME
rm -rf $HERMES_HOME
rm -rf chain_*.log
rm -rf hermes.log
rm -rf $ICA_PACKET_JSON

# Make hermes dir and copy the config
mkdir $HERMES_HOME
cp scripts/hermes_config.toml "$HERMES_HOME/config.toml"

# Build the binary
make build-mac # make build-mac

echo "==> Initializing sourcehub-1..."
$BIN init $C1V1_NAME --chain-id $CHAIN1_ID --default-denom="uopen" --home="$C1V1_HOME"
$BIN keys add $C1V1 --keyring-backend=test --home="$C1V1_HOME"
VALIDATOR1_ADDR=$($BIN keys show $C1V1 -a --keyring-backend=test --home="$C1V1_HOME")
$BIN genesis add-genesis-account $VALIDATOR1_ADDR 1000000000000uopen --home="$C1V1_HOME"
echo "divert tenant reveal hire thing jar carry lonely magic oak audit fiber earth catalog cheap merry print clown portion speak daring giant weird slight" | $BIN keys add source --recover --keyring-backend=test --home="$C1V1_HOME"
SOURCE_ADDR=$($BIN keys show source -a --keyring-backend=test --home="$C1V1_HOME")
$BIN genesis add-genesis-account $SOURCE_ADDR 1000000000000uopen --home="$C1V1_HOME"
$BIN genesis gentx $C1V1 100000000uopen --chain-id $CHAIN1_ID --keyring-backend=test --home="$C1V1_HOME"
$BIN genesis collect-gentxs --home "$C1V1_HOME"
$BIN genesis validate-genesis --home "$C1V1_HOME"

jq '.app_state.transfer.port_id = "transfer"' "$C1V1_HOME/config/genesis.json" > tmp.json && mv tmp.json "$C1V1_HOME/config/genesis.json"
jq '.app_state.transfer += {"params": {"send_enabled": true, "receive_enabled": true}}' "$C1V1_HOME/config/genesis.json" > tmp.json && mv tmp.json "$C1V1_HOME/config/genesis.json"

sed -i '' 's/minimum-gas-prices = ""/minimum-gas-prices = "0.001uopen,0.001ucredit"/' "$C1V1_HOME/config/app.toml"
sed -i '' 's/^timeout_propose = .*/timeout_propose = "500ms"/' "$C1V1_HOME/config/config.toml"
sed -i '' 's/^timeout_prevote = .*/timeout_prevote = "500ms"/' "$C1V1_HOME/config/config.toml"
sed -i '' 's/^timeout_precommit = .*/timeout_precommit = "500ms"/' "$C1V1_HOME/config/config.toml"
sed -i '' 's/^timeout_commit = .*/timeout_commit = "1s"/' "$C1V1_HOME/config/config.toml"

sleep 1

echo "==> Initializing sourcehub-2..."
$BIN init $C2V1_NAME --chain-id $CHAIN2_ID --default-denom="uopen" --home="$C2V1_HOME"
$BIN keys add $C2V1 --keyring-backend=test --home="$C2V1_HOME"
VALIDATOR2_ADDR=$($BIN keys show $C2V1 -a --keyring-backend=test --home="$C2V1_HOME")
$BIN genesis add-genesis-account $VALIDATOR2_ADDR 1000000000000uopen --home="$C2V1_HOME"
echo "divert tenant reveal hire thing jar carry lonely magic oak audit fiber earth catalog cheap merry print clown portion speak daring giant weird slight" | $BIN keys add source --recover --keyring-backend=test --home="$C2V1_HOME"
SOURCE2_ADDR=$($BIN keys show source -a --keyring-backend=test --home="$C2V1_HOME")
$BIN genesis add-genesis-account $SOURCE2_ADDR 1000000000000uopen --home="$C2V1_HOME"
$BIN genesis gentx $C2V1 100000000uopen --chain-id $CHAIN2_ID --keyring-backend=test --home="$C2V1_HOME"
$BIN genesis collect-gentxs --home "$C2V1_HOME"
$BIN genesis validate-genesis --home "$C2V1_HOME"

jq '.app_state.transfer.port_id = "transfer"' "$C2V1_HOME/config/genesis.json" > tmp.json && mv tmp.json "$C2V1_HOME/config/genesis.json"
jq '.app_state.transfer += {"params": {"send_enabled": true, "receive_enabled": true}}' "$C2V1_HOME/config/genesis.json" > tmp.json && mv tmp.json "$C2V1_HOME/config/genesis.json"

sed -i '' 's/minimum-gas-prices = ""/minimum-gas-prices = "0.001uopen,0.001ucredit"/' "$C2V1_HOME/config/app.toml"
sed -i '' 's/^timeout_propose = .*/timeout_propose = "500ms"/' "$C2V1_HOME/config/config.toml"
sed -i '' 's/^timeout_prevote = .*/timeout_prevote = "500ms"/' "$C2V1_HOME/config/config.toml"
sed -i '' 's/^timeout_precommit = .*/timeout_precommit = "500ms"/' "$C2V1_HOME/config/config.toml"
sed -i '' 's/^timeout_commit = .*/timeout_commit = "1s"/' "$C2V1_HOME/config/config.toml"

sleep 1

echo "==> Starting sourcehub-1..."
$BIN start \
  --home $C1V1_HOME \
  --rpc.laddr $C1V1_RPC \
  --rpc.pprof_laddr $C1V1_PPROF \
  --p2p.laddr $C1V1_P2P \
  --grpc.address $C1V1_GRPC \
  --address $C1V1_ADDR \
  > chain_1_1.log 2>&1 &
echo "sourcehub-1 running"

sleep 1

echo "==> Starting sourcehub-2..."
$BIN start \
  --home $C2V1_HOME \
  --rpc.laddr $C2V1_RPC \
  --rpc.pprof_laddr $C2V1_PPROF \
  --p2p.laddr $C2V1_P2P \
  --grpc.address $C2V1_GRPC \
  --address $C2V1_ADDR \
  > chain_2_1.log 2>&1 &
echo "sourcehub-2 running"

sleep 5

# Add hermes key (same for both chains)
echo "divert tenant reveal hire thing jar carry lonely magic oak audit fiber earth catalog cheap merry print clown portion speak daring giant weird slight" > /tmp/mnemonic.txt
hermes keys add --chain $CHAIN1_ID --mnemonic-file /tmp/mnemonic.txt
hermes keys add --chain $CHAIN2_ID --mnemonic-file /tmp/mnemonic.txt
rm /tmp/mnemonic.txt

echo "==> Creating IBC connection (no channel)..."
hermes create connection \
  --a-chain $CHAIN1_ID \
  --b-chain $CHAIN2_ID

sleep 1

echo "==> Starting Hermes..."
hermes start > hermes.log 2>&1 &

sleep 1

# Detect latest connection id on controller if not provided
if [ -z "$CONNECTION_ID" ]; then
  CONNECTION_ID=$($BIN q ibc connection connections --node "$C1V1_RPC" -o json | jq -r '.connections | last | .id')
  echo "Detected CONNECTION_ID=$CONNECTION_ID"
fi

# ICA testing section
CTRL_HOME=$C1V1_HOME
CTRL_RPC=$C1V1_RPC
CTRL_CHAIN_ID=$CHAIN1_ID

echo "==> Registering interchain account on $CTRL_CHAIN_ID (controller) via $CONNECTION_ID"
$BIN tx interchain-accounts controller register "$CONNECTION_ID" \
  --from $SOURCE_ADDR \
  --keyring-backend test \
  --chain-id $CTRL_CHAIN_ID \
  --home "$CTRL_HOME" \
  --node "$CTRL_RPC" \
  --ordering ORDER_ORDERED \
  --version "" \
  --gas auto \
  --fees 500uopen \
  --yes

echo "==> Waiting for ICA handshake to complete..."
sleep 10

echo "==> Querying host ICA address..."
ICA_ADDR=$($BIN q interchain-accounts controller interchain-account $SOURCE_ADDR "$CONNECTION_ID" \
  --node "$CTRL_RPC" -o json | jq -r '.address')
echo "Host ICA address: $ICA_ADDR"

if [ -z "$ICA_ADDR" ] || [ "$ICA_ADDR" = "null" ]; then
  echo "Failed to resolve ICA address. Check Hermes and connection IDs."
  exit 1
fi

echo "==> Preparing ICA packet with ACP MsgCreatePolicy..."
go run ./cmd/ica_packet_gen --creator "$ICA_ADDR" --policy "$POLICY_CONTENT" > $ICA_PACKET_JSON

echo "Packet:
$(cat $ICA_PACKET_JSON)"

echo "==> Sending ICA tx packet from controller..."
$BIN tx interchain-accounts controller send-tx $CONNECTION_ID $ICA_PACKET_JSON \
  --from $SOURCE_ADDR \
  --keyring-backend test \
  --chain-id $CTRL_CHAIN_ID \
  --home "$CTRL_HOME" \
  --node "$CTRL_RPC" \
  --gas auto --gas-adjustment 1.3 \
  --fees 500uopen \
  --yes

echo "Waiting for host execution..."
sleep 5

echo "==> Querying ACP policy IDs on host..."
$BIN q acp policy-ids --node "$C2V1_RPC" -o json || true

echo "DONE"


