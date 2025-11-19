#!/bin/sh
set -e

rm -rf "$HOME/.sourcehub" || true

sedi() {
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "$@"
  else
    sed -i "$@"
  fi
}

CHAIN_ID="sourcehub-dev"
VALIDATOR="validator"
FAUCET="faucet"
NODE_NAME="node"
BIN="build/sourcehubd"
GENESIS="$HOME/.sourcehub/config/genesis.json"
FAUCET_KEY="$HOME/.sourcehub/config/faucet-key.json"
APP_TOML="$HOME/.sourcehub/config/app.toml"
CONFIG_TOML="$HOME/.sourcehub/config/config.toml"

$BIN init $NODE_NAME --chain-id $CHAIN_ID --default-denom="uopen"

# Copy faucet key to config and add it to the keyring
mkdir -p "$HOME/.sourcehub/config" && cp scripts/faucet-key.json "$FAUCET_KEY"
FAUCET_MNEMONIC=$(jq -r '.mnemonic' "$FAUCET_KEY")
echo "$FAUCET_MNEMONIC" | $BIN keys add $FAUCET --recover --keyring-backend test
FAUCET_ADDR=$($BIN keys show $FAUCET -a --keyring-backend test)

$BIN keys add $VALIDATOR --keyring-backend test
VALIDATOR_ADDR=$($BIN keys show $VALIDATOR -a --keyring-backend test)
$BIN genesis add-genesis-account $VALIDATOR_ADDR 1000000000000000uopen # 1b open
$BIN genesis add-genesis-account $FAUCET_ADDR 100000000000000uopen,1000000000000000ucredit # 100m open and 1b ucredit
$BIN genesis gentx $VALIDATOR 100000000000000uopen --chain-id $CHAIN_ID --keyring-backend test # 100m open
$BIN genesis collect-gentxs

# Enable IBC
jq '.app_state.transfer.port_id = "transfer"' "$GENESIS" > tmp.json && mv tmp.json "$GENESIS"
jq '.app_state.transfer += {"params": {"send_enabled": true, "receive_enabled": true}}' "$GENESIS" > tmp.json && mv tmp.json "$GENESIS"

# Enable/disable zero-fee transactions
jq '.app_state.hub.chain_config.allow_zero_fee_txs = true' "$GENESIS" > tmp.json && mv tmp.json "$GENESIS"
jq '.app_state.hub.chain_config.ignore_bearer_auth = true' "$GENESIS" > tmp.json && mv tmp.json "$GENESIS"

# app.toml
sedi 's/minimum-gas-prices = .*/minimum-gas-prices = "0.001uopen,0.001ucredit"/' "$APP_TOML"
sedi 's/^enabled = .*/enabled = true/' "$APP_TOML"
sedi 's/^prometheus-retention-time = .*/prometheus-retention-time = 60/' "$APP_TOML"
sedi 's/^enabled-unsafe-cors = .*/enabled-unsafe-cors = true/' "$APP_TOML"
sedi 's/^enable = .*/enable = true/' "$APP_TOML"
sedi 's/^swagger = .*/swagger = true/' "$APP_TOML"
sedi 's/^enable_faucet = .*/enable_faucet = true/' "$APP_TOML"

# config.toml
sedi 's/^timeout_propose = .*/timeout_propose = "500ms"/' "$CONFIG_TOML"
sedi 's/^timeout_prevote = .*/timeout_prevote = "500ms"/' "$CONFIG_TOML"
sedi 's/^timeout_precommit = .*/timeout_precommit = "500ms"/' "$CONFIG_TOML"
sedi 's/^timeout_commit = .*/timeout_commit = "1s"/' "$CONFIG_TOML"
sedi 's/^prometheus = .*/prometheus = true/' "$CONFIG_TOML"
sedi 's/^cors_allowed_origins = .*/cors_allowed_origins = ["*"]/' "$CONFIG_TOML"

echo "Validator Address $VALIDATOR_ADDR"
