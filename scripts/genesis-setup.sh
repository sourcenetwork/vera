#!/bin/sh
set -e

rm -rf ~/.sourcehub || true

sedi() {
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "$@"
  else
    sed -i "$@"
  fi
}

CHAIN_ID="sourcehub-dev"
VALIDATOR="validator"
NODE_NAME="node"
BIN="build/sourcehubd"

$BIN init $NODE_NAME --chain-id $CHAIN_ID --default-denom="uopen"
$BIN keys add $VALIDATOR --keyring-backend test
VALIDATOR_ADDR=$($BIN keys show $VALIDATOR -a --keyring-backend test)
$BIN genesis add-genesis-account $VALIDATOR_ADDR 1000000000000000uopen # 1b open
$BIN genesis gentx $VALIDATOR 100000000000000uopen --chain-id $CHAIN_ID --keyring-backend test # 100m open
$BIN genesis collect-gentxs

# Enable IBC
jq '.app_state.transfer.port_id = "transfer"' ~/.sourcehub/config/genesis.json > tmp.json && mv tmp.json ~/.sourcehub/config/genesis.json
jq '.app_state.transfer += {"params": {"send_enabled": true, "receive_enabled": true}}' ~/.sourcehub/config/genesis.json > tmp.json && mv tmp.json ~/.sourcehub/config/genesis.json

# Enable/disable zero-fee transactions
jq '.app_state.app_params.allow_zero_fee_txs = false' ~/.sourcehub/config/genesis.json > tmp.json && mv tmp.json ~/.sourcehub/config/genesis.json

# app.toml
sedi 's/minimum-gas-prices = .*/minimum-gas-prices = "0.001uopen,0.001ucredit"/' ~/.sourcehub/config/app.toml
sedi 's/^enabled = .*/enabled = true/' ~/.sourcehub/config/app.toml
sedi 's/^prometheus-retention-time = .*/prometheus-retention-time = 60/' ~/.sourcehub/config/app.toml

# config.toml
sedi 's/^timeout_propose = .*/timeout_propose = "500ms"/' ~/.sourcehub/config/config.toml
sedi 's/^timeout_prevote = .*/timeout_prevote = "500ms"/' ~/.sourcehub/config/config.toml
sedi 's/^timeout_precommit = .*/timeout_precommit = "500ms"/' ~/.sourcehub/config/config.toml
sedi 's/^timeout_commit = .*/timeout_commit = "1s"/' ~/.sourcehub/config/config.toml
sedi 's/^prometheus = .*/prometheus = true/' ~/.sourcehub/config/config.toml

echo "Validator Address $VALIDATOR_ADDR"
