# SourceHub

Source's Trust Layer

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
# Run build/sourcehubd directly
./scripts/dev-entrypoint.sh start
# or using Docker
docker-compose up
```

The `./scripts/dev-entrypoint.sh` script runs `./scripts/genesis-setup.sh` internally and automatically initializes a new node with chain ID `sourcehub-dev`, creates validator and faucet keys, and configures dev settings.

### Configuration

The `./scripts/genesis-setup.sh` script configures:

- **Zero-fee transactions**: `allow_zero_fee_txs = true` for easier testing
- **Gas**: Minimum gas prices set to `0.001uopen,0.001ucredit`
- **IBC**: IBC transfers enabled
- **API & Swagger**: API and Swagger enabled
- **CORS**: CORS enabled to interact with API locally
- **Metrics**: Prometheus metrics enabled
- **Faucet**: Built-in faucet with test funds

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
