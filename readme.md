# Vera

Source's Trust Layer

## Getting Started

To quickly spin up a standalone Vera network, we recommend using Docker.
This repository contains a Dockerfile which if run with the environment variable `STANDALONE=1`, spins up a new single node Vera network.

To get started build the docker image with,

```
docker image build -t vera:latest .
```

and start it with:

```
docker run -p 9090:9090 -p 26657:26657 -p 26656:26656 -p 1317:1317 -e STANDALONE=1 vera:latest
```

The container will start a new network with no fees and a funded faucet account which can be used.
The funded account is static for all new instances of the standalone deployment and can be imported by a keyring outside docker or a wallet (eg keplr) to broadcast transactions.

The account mnemonic is:

```
comic very pond victory suit tube ginger antique life then core warm loyal deliver iron fashion erupt husband weekend monster sunny artist empty uphold
```

For a complete list of params regarding the Dockerfile, see [docker readme](./docker/README.md)

## Development

### Prerequisites

- Go 1.23 or later
- Ignite CLI (for proto generation)
- Docker (optional)

### Building the Project

```bash
# Generate protos
make proto
# Build the binary
make build # or "make build-mac" for macOS
# Install globally
make install
```

### Running the Chain Locally

```bash
# Run build/verad directly
./scripts/dev-entrypoint.sh start
# or using Docker
docker-compose up
```

The `./scripts/dev-entrypoint.sh` script runs `./scripts/genesis-setup.sh` internally and automatically initializes a new node with chain ID `vera-dev`, creates validator and faucet keys, and configures dev settings.

### Configuration

The `./scripts/genesis-setup.sh` script configures:

- **Zero-fee transactions**: `allow_zero_fee_txs = true` for easier testing
- **Gas**: Minimum gas prices set to `0.001uopen,0.001ucredit`
- **IBC**: IBC transfers enabled
- **API & Swagger**: API and Swagger enabled
- **CORS**: CORS enabled to interact with API locally
- **Metrics**: Prometheus metrics enabled
- **Faucet**: Built-in faucet with test funds

Nodes use `$HOME/.vera` by default.

### Testing

```bash
# Run all tests
make test
# Run test matrix
make test:all
```

## Documentation

- [Ignite CLI](https://ignite.com/cli)
- [Tutorials](https://docs.ignite.com/guide)
- [Ignite CLI docs](https://docs.ignite.com)
- [Cosmos SDK docs](https://docs.cosmos.network)
