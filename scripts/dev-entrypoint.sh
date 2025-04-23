#!/bin/bash

if [ ! -e "~/INITIALIZED" ]; then
    scripts/genesis-setup.sh
    touch "~/INITIALIZED"
fi

# Set VALIDATOR_ADDR env var here

exec build/sourcehubd $@
