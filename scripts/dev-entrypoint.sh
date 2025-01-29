#!/usr/bin/bash

if [ ! -e "~/INITIALIZED" ]; then
    scripts/genesis-setup.sh
    sed -i 's/^timeout_commit = .*/timeout_commit = "1s"/' ~/.sourcehub/config/config.toml
    touch "~/INITIALIZED"
fi

# Set VALIDATOR_ADDR env var here

exec /app/build/sourcehubd $@
