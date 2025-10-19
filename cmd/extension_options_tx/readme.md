# Testing JWS extension option with DID-based feegrant

This document explains how to test the JWS extension option functionality with DID-based feegrants.

## Prerequisites

- sourcehub node running locally (e.g. `./scripts/dev-entrypoint.sh start`)
- Faucet account and validator account set up in keyring (set by default when running `dev-entrypoint.sh` script)
- Test policy file available in `./scripts/test-policy.yaml`

## Steps

### 1. Grant DID-based fee allowance

Add a DID-based fee grant from the faucet address to the DID used in the script:

```bash
build/sourcehubd tx feegrant grant-did source12d9hjf0639k995venpv675sju9ltsvf8u5c9jt did:key:z6MknVX5y2APs6LH21s9FusVozvdKKwDhFAqq3jzwAr6v21a --spend-limit 1000000uopen --keyring-backend test --chain-id=sourcehub-dev --gas auto --fees 200uopen -y
```

### 2. Verify the feegrant (optional)

Check that the DID allowance was added correctly:

```bash
build/sourcehubd q feegrant did-grant source12d9hjf0639k995venpv675sju9ltsvf8u5c9jt did:key:z6MknVX5y2APs6LH21s9FusVozvdKKwDhFAqq3jzwAr6v21a
```

### 3. Run the extension options script

Execute the script to create a policy with validator as the sender and faucet as the fee payer:

```bash
go run ./cmd/extension_options_tx scripts/test-policy.yaml
```

### 4. Confirm policy creation (optional)

Check that the policy was created successfully:

```bash
build/sourcehubd q acp policy-ids
```

## How it works

- The script creates a transaction signed by the **validator** account
- The transaction includes a **JWS extension option** with a signature from the DID
- The **faucet** account pays the transaction fees via DID-based feegrant
- The policy is created with the validator as the creator

## Expected behavior

1. JWS extension option is validated and DID is extracted.
2. DID-based feegrant is used to pay fees.
3. Transaction succeeds and policy is created.
4. Events show `use_feegrant` with the correct DID.

## Current limitations

> **NOTE**: The current create/edit policy logic uses the msg creator address as the policy owner. We should make changes to be able to use DID from the JWS as the owner instead.