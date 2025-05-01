#!/bin/sh

set -e

BIN="build/sourcehubd"

CHAIN1_ID="sourcehub-1"
CHAIN2_ID="sourcehub-2"

C1V1_HOME="$HOME/.sourcehub-1-1"
C2V1_HOME="$HOME/.sourcehub-2-1"
C1V2_HOME="$HOME/.sourcehub-1-2"
C2V2_HOME="$HOME/.sourcehub-2-2"
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

C1V2_GRPC=localhost:9093
C1V2_P2P=localhost:27671
C1V2_ADDR=tcp://0.0.0.0:27672
C1V2_RPC=tcp://127.0.0.1:26673
C1V2_PPROF=localhost:6063

C2V2_GRPC=localhost:9094
C2V2_P2P=localhost:27674
C2V2_ADDR=tcp://0.0.0.0:27675
C2V2_RPC=tcp://127.0.0.1:26676
C2V2_PPROF=localhost:6064

C1V1_NAME="validator1-1-node"
C2V1_NAME="validator2-1-node"
C1V2_NAME="validator1-2-node"
C2V2_NAME="validator2-2-node"

C1V1="validator1-1"
C2V1="validator2-1"
C1V2="validator1-2"
C2V2="validator2-2"

C1V2_JSON="scripts/validator1-2.json"
C2V2_JSON="scripts/validator2-2.json"

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
rm -rf $C1V2_HOME
rm -rf $C2V2_HOME
rm -rf chain_*.log
rm -rf hermes.log
rm $C1V2_JSON 2>/dev/null || true
rm $C2V2_JSON 2>/dev/null || true

# Make hermes dir and copy the config
mkdir $HERMES_HOME
cp scripts/hermes_config.toml "$HERMES_HOME/config.toml"

# Build the binary
make build # make build-mac

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
SOURCE_ADDR=$($BIN keys show source -a --keyring-backend=test --home="$C2V1_HOME")
$BIN genesis add-genesis-account $SOURCE_ADDR 1000000000000uopen --home="$C2V1_HOME"
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

# Let chains start before adding more validators
sleep 5

echo "==> Initializing second validator on sourcehub-1..."
$BIN init $C1V2_NAME --chain-id $CHAIN1_ID --home="$C1V2_HOME"
$BIN keys add $C1V2 --keyring-backend=test --home="$C1V2_HOME"
VAL11_ADDR=$($BIN keys show validator1-2 -a --keyring-backend=test --home="$C1V2_HOME")

rsync -a --exclude priv_validator_state.json ~/.sourcehub-1-1/data/ ~/.sourcehub-1-2/data/
VAL1_ID=$($BIN tendermint show-node-id --home="$C1V1_HOME")
sed -i '' "s|^persistent_peers = .*|persistent_peers = \"$VAL1_ID@$C1V1_P2P\"|" "$C1V2_HOME/config/config.toml"

sed -i '' 's/minimum-gas-prices = ""/minimum-gas-prices = "0.001uopen,0.001ucredit"/' "$C1V2_HOME/config/app.toml"
sed -i '' 's/^timeout_propose = .*/timeout_propose = "500ms"/' "$C1V2_HOME/config/config.toml"
sed -i '' 's/^timeout_prevote = .*/timeout_prevote = "500ms"/' "$C1V2_HOME/config/config.toml"
sed -i '' 's/^timeout_precommit = .*/timeout_precommit = "500ms"/' "$C1V2_HOME/config/config.toml"
sed -i '' 's/^timeout_commit = .*/timeout_commit = "1s"/' "$C1V2_HOME/config/config.toml"

echo "==> Funding validator 2 account on sourcehub-1..."
$BIN tx bank send source $VAL11_ADDR 10000000000uopen \
  --from source \
  --keyring-backend test \
  --home $C1V1_HOME \
  --chain-id $CHAIN1_ID \
  --node $C1V1_RPC \
  --gas auto \
  --fees 500uopen \
  --yes

sleep 1

echo "==> Creating validator 2 on sourcehub-1..."
PUBKEY_JSON1=$($BIN tendermint show-validator --home="$C1V2_HOME")
echo "{
  \"pubkey\": $PUBKEY_JSON1,
  \"amount\": \"1000000000uopen\",
  \"moniker\": \"$C1V2_NAME\",
  \"identity\": \"\",
  \"website\": \"\",
  \"security\": \"\",
  \"details\": \"\",
  \"commission-rate\": \"0.1\",
  \"commission-max-rate\": \"0.2\",
  \"commission-max-change-rate\": \"0.01\",
  \"min-self-delegation\": \"1\"
}" > $C1V2_JSON

$BIN tx staking create-validator "$C1V2_JSON" \
  --from $C1V2 \
  --chain-id $CHAIN1_ID \
  --home $C1V2_HOME \
  --keyring-backend=test \
  --node $C1V1_RPC \
  --gas auto \
  --fees 500uopen \
  --yes

sleep 1

echo "==> Starting validator 2 on sourcehub-1..."
$BIN start \
  --home $C1V2_HOME \
  --rpc.laddr $C1V2_RPC \
  --rpc.pprof_laddr $C1V2_PPROF \
  --p2p.laddr $C1V2_P2P \
  --grpc.address $C1V2_GRPC \
  --address $C1V2_ADDR \
  > chain_1_2.log 2>&1 &

sleep 1

echo "==> Initializing validator 2 on sourcehub-2..."
$BIN init $C2V2_NAME --chain-id $CHAIN2_ID --home="$C2V2_HOME"
$BIN keys add $C2V2 --keyring-backend=test --home="$C2V2_HOME"
VAL22_ADDR=$($BIN keys show validator2-2 -a --keyring-backend=test --home="$C2V2_HOME")

rsync -a --exclude priv_validator_state.json ~/.sourcehub-2-1/data/ ~/.sourcehub-2-2/data/
VAL2_ID=$($BIN tendermint show-node-id --home="$C2V1_HOME")
sed -i '' "s|^persistent_peers = .*|persistent_peers = \"$VAL2_ID@$C2V1_P2P\"|" "$C2V2_HOME/config/config.toml"

sed -i '' 's/minimum-gas-prices = ""/minimum-gas-prices = "0.001uopen,0.001ucredit"/' "$C2V2_HOME/config/app.toml"
sed -i '' 's/^timeout_propose = .*/timeout_propose = "500ms"/' "$C2V2_HOME/config/config.toml"
sed -i '' 's/^timeout_prevote = .*/timeout_prevote = "500ms"/' "$C2V2_HOME/config/config.toml"
sed -i '' 's/^timeout_precommit = .*/timeout_precommit = "500ms"/' "$C2V2_HOME/config/config.toml"
sed -i '' 's/^timeout_commit = .*/timeout_commit = "1s"/' "$C2V2_HOME/config/config.toml"

echo "==> Funding validato 2 account on sourcehub-2..."
$BIN tx bank send source $VAL22_ADDR 10000000000uopen \
  --from source \
  --keyring-backend test \
  --home $C2V1_HOME \
  --chain-id $CHAIN2_ID \
  --node $C2V1_RPC \
  --gas auto \
  --fees 500uopen \
  --yes

sleep 1

echo "==> Creating validator 2 on sourcehub-2..."
PUBKEY_JSON2=$($BIN tendermint show-validator --home="$C2V2_HOME")
echo "{
  \"pubkey\": $PUBKEY_JSON2,
  \"amount\": \"1000000000uopen\",
  \"moniker\": \"$C2V2_NAME\",
  \"identity\": \"\",
  \"website\": \"\",
  \"security\": \"\",
  \"details\": \"\",
  \"commission-rate\": \"0.1\",
  \"commission-max-rate\": \"0.2\",
  \"commission-max-change-rate\": \"0.01\",
  \"min-self-delegation\": \"1\"
}" > $C2V2_JSON

$BIN tx staking create-validator "$C2V2_JSON" \
  --from $C2V2 \
  --chain-id $CHAIN2_ID \
  --home $C2V2_HOME \
  --keyring-backend=test \
  --node $C2V1_RPC \
  --gas auto \
  --fees 500uopen \
  --yes

sleep 1

echo "==> Starting validator 2 on sourcehub-2..."
$BIN start \
  --home $C2V2_HOME \
  --rpc.laddr $C2V2_RPC \
  --rpc.pprof_laddr $C2V2_PPROF \
  --p2p.laddr $C2V2_P2P \
  --grpc.address $C2V2_GRPC \
  --address $C2V2_ADDR \
  > chain_2_2.log 2>&1 &

# Let chains run for a few seconds to make sure all validators are in sync
sleep 10

# Add hermes key (same for both chains)
echo "divert tenant reveal hire thing jar carry lonely magic oak audit fiber earth catalog cheap merry print clown portion speak daring giant weird slight" > /tmp/mnemonic.txt
hermes keys add --chain $CHAIN1_ID --mnemonic-file /tmp/mnemonic.txt
hermes keys add --chain $CHAIN2_ID --mnemonic-file /tmp/mnemonic.txt
rm /tmp/mnemonic.txt

echo "==> Creating channel..."
hermes create channel \
  --a-chain $CHAIN1_ID \
  --b-chain $CHAIN2_ID \
  --a-port transfer \
  --b-port transfer \
  --new-client-connection \
  --yes

sleep 1

echo "==> Starting Hermes..."
hermes start > hermes.log 2>&1 &

sleep 5

$BIN tx ibc-transfer transfer transfer channel-0 \
  $SOURCE_ADDR 1000uopen \
  --from source \
  --keyring-backend test \
  --chain-id sourcehub-1 \
  --home "$C1V1_HOME" \
  --node "$C1V1_RPC" \
  --gas auto --gas-adjustment 1.3 \
  --fees 500uopen \
  --yes

sleep 5

$BIN q bank balances $SOURCE_ADDR --node "$C2V1_RPC"

echo "DONE"
