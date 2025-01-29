#!/usr/bin/sh
set -e

CHAIN_ID="sourcehub-dev"
VALIDATOR="validator"
NODE_NAME="node"
BIN="build/sourcehubd"

rm -rf ~/.sourcehub || true

$BIN init $NODE_NAME --chain-id $CHAIN_ID

$BIN keys add $VALIDATOR --keyring-backend test
VALIDATOR_ADDR=$($BIN keys show $VALIDATOR -a --keyring-backend test)
$BIN genesis add-genesis-account $VALIDATOR_ADDR 100000000000open
$BIN genesis gentx $VALIDATOR 100000000open --chain-id $CHAIN_ID --keyring-backend test

$BIN genesis collect-gentxs

sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0open"/' ~/.sourcehub/config/app.toml

echo "Validator Address $VALIDATOR_ADDR"
