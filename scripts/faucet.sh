#!/usr/bin/env sh
#
FAUCET_KEY_FILE="$HOME/.sourcehub/config/faucet-key.json"

if [ ! -f "$FAUCET_KEY_FILE" ]; then
    echo "Error: Faucet key file not found at $FAUCET_KEY_FILE"
    exit 1
fi

FAUCET_ADDRESS=$(jq -r '.address' "$FAUCET_KEY_FILE")
if [ -z "$FAUCET_ADDRESS" ] || [ "$FAUCET_ADDRESS" = "null" ]; then
    echo "Error: Could not extract faucet address from $FAUCET_KEY_FILE"
    exit 1
fi

if [ -z $1 ];
then
    echo 'faucet.sh target-account [amount]'
    exit 1
fi

TARGET_ADDRESS=$1
AMOUNT=${2:-"1000000000uopen"}

echo "Sending $AMOUNT from faucet ($FAUCET_ADDRESS) to $TARGET_ADDRESS"

build/sourcehubd tx bank send "$FAUCET_ADDRESS" "$TARGET_ADDRESS" "$AMOUNT" --from "$FAUCET_ADDRESS" --chain-id sourcehub-dev --keyring-backend test --gas auto --fees 200uopen -y