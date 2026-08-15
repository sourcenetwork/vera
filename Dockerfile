FROM golang:1.26.2 AS builder

WORKDIR /app

# Cache deps
COPY go.* /app/
RUN go mod download

# Build
COPY . /app
RUN --mount=type=cache,target=/root/.cache go build -o /app/build/verad ./cmd/verad

# Deployment entrypoint
FROM debian:bookworm-slim

COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
COPY docker/faucet-key.json /etc/vera/faucet-key.json
COPY --from=builder /app/build/verad /usr/local/bin/verad
# Copy the default config files to override the container with
COPY docker/configs/*.toml /etc/vera/

RUN useradd --create-home --home-dir /home/node node && mkdir /vera && chown node:node /vera && ln -s /vera /home/node/.vera && chown node:node -R /home/node && chmod -R 555 /etc/vera

# MONIKER sets the node moniker
ENV MONIKER="node"
# CHAIN_ID sets the id for the chain which will be initialized
ENV CHAIN_ID="vera-dev"

# GENESIS_PATH is an optional variable which if set must point to a genesis file mounted in the container.
# The file is copied to the configuration directory during the first container initialization
# If empty, the entrypoint will generate a new genesis
ENV GENESIS_PATH=""

# MNEMONIC_PATH is an optional varible which, if set, must point to a file containing a 
# cosmos key mnemonic. The mnemonic will be used to restore the node operator / validator key.
# If empty, the entrypoint will generate a new key
ENV MNEMONIC_PATH=""

# CONSENSUS_KEY_PATH is an optional variable which, if set, must point to a file containg
# a comebft consesus key for the validator.
# If empty, the entrypoint will generate a new key
ENV CONSENSUS_KEY_PATH=""

# COMET_NODE_KEY_PATH is an optional variable which, if set, must point to a file containg
# a comebft p2p node key.
# If empty, the entrypoint will generate a new key
ENV COMET_NODE_KEY_PATH=""

# COMET_CONFIG_PATH is an optional variable which, if set, will overwrite
# the default cofig.toml with the provided file.
ENV COMET_CONFIG_PATH=""

# APP_CONFIG_PATH is an optional variable which, if set, will overwrite
# the default app.toml with the provided file.
ENV APP_CONFIG_PATH=""

ENV STANDALONE=""

# Comet P2P Port
EXPOSE 26656

# Comet RPC Port
EXPOSE 26657

# Vera GRPC Port
EXPOSE 9090

# Vera HTTP API Port
EXPOSE 1317

USER node
VOLUME ["/vera"]
ENTRYPOINT ["entrypoint.sh"]
CMD ["verad", "start"]
