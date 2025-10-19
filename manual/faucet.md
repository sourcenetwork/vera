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

Sends 1000000000uopen (i.e. 1,000 $OPEN) from the faucet to the specified address.

### Initialize Account
```bash
curl -X POST http://localhost:1317/faucet/init-account \
  -H "Content-Type: application/json" \
  -d '{
    "address": "source1..."
  }'
```

Initializes an account in the auth module by sending 1 uopen. This ensures the account exists on-chain.

### Grant Fee Allowance
```bash
curl -X POST http://localhost:1317/faucet/grant-allowance \
  -H "Content-Type: application/json" \
  -d '{
    "address": "source1...",
    "amount_limit": {
      "denom": "uopen",
      "amount": "10000000000000"
    },
    "expiration": "2025-12-31T23:59:59Z"
  }'
```

Grants a fee allowance from the faucet account to the specified address. This allows the grantee to pay transaction fees using the granter's (e.g. faucet) account balance.

**Notes:**
- `amount_limit` is optional. If not provided, defaults to 10,000 OPEN (10000000000 uopen)
- `expiration` is optional. If not provided, defaults to 30 days from now

### Grant DID Fee Allowance
```bash
curl -X POST http://localhost:1317/faucet/grant-did-allowance \
  -H "Content-Type: application/json" \
  -d '{
    "did": "did:key:alice",
    "amount_limit": {
      "denom": "uopen",
      "amount": "10000000000"
    },
    "expiration": "2025-12-31T23:59:59Z"
  }'
```

Grants a fee allowance from the faucet account to the specified DID. This allows transactions with JWS extensions containing the DID to pay transaction fees using the faucet's account balance.

**Notes:**
- `amount_limit` is optional. If not provided, defaults to 10,000 OPEN (10000000000 uopen)
- `expiration` is optional. If not provided, defaults to 30 days from now

**Response:**
```json
{
  "message": "DID fee allowance granted successfully",
  "txhash": "A1B2C3...",
  "granter": "source12d9hjf0639k995venpv675sju9ltsvf8u5c9jt",
  "grantee_did": "did:key:alice",
  "amount_limit": {
    "denom": "uopen",
    "amount": "10000000000"
  },
  "expiration": "2025-12-31T23:59:59Z"
}
```

**Query DID allowance via CLI:**
```bash
# Query specific DID allowance
build/sourcehubd q feegrant did-grant $(curl -s http://localhost:1317/faucet/info | jq -r '.address') did:key:alice

# List all DID allowances by granter
build/sourcehubd q feegrant did-grants-by-granter $(curl -s http://localhost:1317/faucet/info | jq -r '.address')
```

## Configuration

The faucet can be enabled/disabled in `app.toml`:

```toml
[faucet]
# Defines if the faucet service should be enabled.
enable_faucet = true
```

When disabled (`enable_faucet = false`), the faucet routes are not registered and endpoints return `Not Implemented`.

The faucet is disabled by default and must be explicitly enabled in configuration (or via `./scripts/genesis-setup.sh`) .

## CLI Usage

The CLI faucet script (`./scripts/faucet.sh`) operates independently of the `enable_faucet` setting in `app.toml` and uses the `./scripts/faucet-key.json` directly.

```bash
# Send 1,000 $OPEN to an address
./scripts/faucet.sh source1...
# Send custom amount
./scripts/faucet.sh source1... 5000000000uopen
```

## Security

- For development and testing purposes only.
- Uses mnemonic from `scripts/faucet-key.json` that is copied into node config directory (e.g. `$HOME/.sourcehub/config`).
- `/faucet/request` is limited to one request of 1,000 $OPEN per address.
- `./scripts/faucet.sh` is not limited and can be used to request arbitrary token amounts.