# Faucet

The SourceHub chain includes a faucet for requesting test funds.

## Setup

Faucet account is initialized as part of the main setup script:

```bash
./scripts/genesis-setup.sh
```

The faucet account is created with 100m $OPEN using a mnemonic from `scripts/faucet-key.json`.

## API Endpoints

### Get Faucet Info
```bash
curl http://localhost:1317/faucet/info
```

Returns faucet address, balance, and request count.

### Request Funds
```bash
curl -X POST http://localhost:1317/faucet/request \
  -H "Content-Type: application/json" \
  -d '{
    "address": "source1..."
  }'
```

Sends 1000000000uopen (i.e. 1000 $OPEN) from the faucet to the specified address.

### Initialize Account
```bash
curl -X POST http://localhost:1317/faucet/init-account \
  -H "Content-Type: application/json" \
  -d '{
    "address": "source1..."
  }'
```

Initializes an account in the auth module by sending 1 uopen. This ensures the account exists on-chain.

## Configuration

The faucet can be enabled/disabled in `app.toml`:

```toml
[faucet]
# Defines if the faucet service should be enabled.
enable_faucet = true
```

When disabled (`enable_faucet = false`), the faucet routes are not registered and endpoints return `Not Implemented`.

The faucet is disabled by default and must be explicitly enabled in configuration (or via `./scripts/genesis-setup.sh`) .

## Security

- For development and testing purposes only.
- Uses hardcoded mnemonic from `scripts/faucet-key.json`.
- `/faucet/request` is limited to one request per address (i.e. 1000 $OPEN).
- `./scripts/faucet.sh` is not limited and can accept arbitrary amounts.