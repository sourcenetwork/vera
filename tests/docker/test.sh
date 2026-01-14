#!/usr/bin/bash
set -e

docker run --rm \
    -e MNEMONIC_PATH=/secrets/mnemonic \
    -e CONSENSUS_KEY_PATH=/secrets/priv_validator_key.json \
    -e COMET_NODE_KEY_PATH=/secrets/node_key.json \
    -e GENESIS_PATH=/secrets/genesis.json \
    -v .:/secrets \
    --name sourcehub-df-test \
    --user 1000:1000 \
    --read-only \
    ghcr.io/sourcenetwork/sourcehub:dev &

sleep 7

last_block=$(docker exec sourcehub-df-test sourcehubd status | jq .sync_info.latest_block_height)

echo "Last SourceHub block $last_block"

test "$last_block" -gt "0"

docker stop sourcehub-df-test