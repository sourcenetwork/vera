#!/bin/bash

set -e 

DEFAULT_CHAIN_ID="sourcehub"
DEFAULT_MONIKER="node"
DEV_FACUET_MNEMONIC="comic very pond victory suit tube ginger antique life then core warm loyal deliver iron fashion erupt husband weekend monster sunny artist empty uphold"

if [ ! -d /sourcehub/.initialized ]; then
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

    # initialize chain in standalone
    if [ "$STANDALONE" = "1" ]; then 
        echo "Standalone mode: generating new genesis"
        # initialize chain / create genesis
        sourcehubd keys add validator --keyring-backend test
        echo $DEV_FACUET_MNEMONIC | sourcehubd keys add faucet --keyring-backend test --recover
        VALIDATOR_ADDR=$(sourcehubd keys show validator -a --keyring-backend test)
        FAUCET_ADDR=$(sourcehubd keys show faucet -a --keyring-backend test)
        sourcehubd genesis add-genesis-account $VALIDATOR_ADDR 1000000000000000uopen # 1b open
        sourcehubd genesis add-genesis-account $FAUCET_ADDR 100000000000000uopen # 100m open
        sourcehubd genesis gentx validator 1000000000000000uopen --chain-id $CHAIN_ID --keyring-backend test # 1b open
        sourcehubd genesis collect-gentxs
        sed -i 's/"allow_zero_fee_txs": false,/"allow_zero_fee_txs": true,/' /sourcehub/config/genesis.json
        cp /etc/sourcehub/faucet-key.json /sourcehub/config/faucet-key.json
        echo "initialized sourcehub genesis"
    else 
        if [ -z "$GENESIS_PATH" ]; then
            echo "GENESIS_PATH not set and standalone is false: provide a genesis file or set env STANDALONE=1"
            exit 1
        fi
        cp $GENESIS_PATH /sourcehub/config/genesis.json
        echo "Loaded Genesis from $GENESIS_PATH"
    fi

    touch /sourcehub/.initialized
else
    echo "Skipping initialization: container previously initialized"
fi

if [ -n "$COMET_CONFIG_PATH" ]; then 
    echo "COMET_CONFIG_PATH set: updating comet config with $COMET_CONFIG_PATH"
    cp $COMET_CONFIG_PATH /sourcehub/config/config.toml
fi

if [ -n "$APP_CONFIG_PATH" ]; then 
    echo "APP_CONFIG_PATH set: updating app config with $APP_CONFIG_PATH"
    cp $APP_CONFIG_PATH /sourcehub/config/app.toml
fi

exec $@