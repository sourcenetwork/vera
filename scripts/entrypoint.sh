#!/bin/bash

set -e 

DEFAULT_CHAIN_ID="sourcehub"
DEFAULT_MONIKER="node"

if [ ! -d /sourcehub/config ]; then
    echo "Initializing SourceHub"

    if [ -z "$CHAIN_ID" ]; then 
        echo "CHAIN_ID not set: using default"
        CHAIN_ID=$DEFAULT_CHAIN_ID
    fi

    if [ -z "$MONIKER" ]; then 
        echo "MONIKER not set: using default"
        MONIKER=$DEFAULT_MONIKER
    fi

    sourcehubd init "$MONIKER" --chain-id $CHAIN_ID --default-denom="uopen" 2>/dev/null

    # recover genesis
    if [ -n "$GENESIS_PATH" ]; then
        echo "GENESIS_PATH set: copying genesis"
        cp $GENESIS_PATH /sourcehub/config/genesis.json
    fi

    # recover account mnemonic
    if [ -n "$MNEMONIC_PATH" ]; then
        echo "MNEMONIC_PATH set: recovering key"
        sourcehubd keys add validator --recover --source $MNEMONIC_PATH --keyring-backend test
    fi

    # if consensus key is set, we recover the full
    # node, including p2p and consensus key
    if [ -n "$CONSENSUS_KEY_PATH" ]; then
        echo "CONSENSUS_KEY_PATH set: recovering validator"
        test -s $CONSENSUS_KEY_PATH || (echo "error: consensus key file is empty" && exit 1)
        test -s $COMET_NODE_KEY_PATH || (echo "error: comet node key file is empty" && exit 1)

        cp $CONSENSUS_KEY_PATH /sourcehub/config/priv_validator_key.json
        cp $COMET_NODE_KEY_PATH /sourcehub/config/node_key.json
    fi
else
    echo "Skipping initialization: container previously initialized"
fi

if [ -n "$COMET_CONFIG_PATH" ]; then 
    echo "COMET_CONFIG_PATH set: updating comet config with $COMET_CONFIG_PATH"
    cp $COMET_CONFIG_PATH /sourcehub/config/config.toml
fi

if [ -n "$APP_CONFIG_PATH" ]; then 
    echo "APP_CONFIG_PATH set: updating app config with $APP_CONFIG_PATH"
    cp $APP_CONFIG_PATH /sourcehub/app.toml
fi

exec $@