#!/usr/bin/sh
set -e

CHAIN_ID="sourcehub-dev"
VALIDATOR="validator"
NODE_NAME="node"
BIN="build/sourcehubd"

rm -rf ~/.sourcehub || true

$BIN init $NODE_NAME --chain-id $CHAIN_ID --default-denom="uopen"

$BIN keys add $VALIDATOR --keyring-backend test
VALIDATOR_ADDR=$($BIN keys show $VALIDATOR -a --keyring-backend test)
$BIN genesis add-genesis-account $VALIDATOR_ADDR 1000000000000000uopen # 1b open
$BIN genesis gentx $VALIDATOR 100000000000000uopen --chain-id $CHAIN_ID --keyring-backend test # 100m open

$BIN genesis collect-gentxs

sed -i 's/^timeout_commit = .*/timeout_commit = "1s"/' ~/.sourcehub/config/config.toml
sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0.001uopen,0.001ucredit"/' ~/.sourcehub/config/app.toml

echo "Validator Address $VALIDATOR_ADDR"
