#!/usr/bin/sh
set -e

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