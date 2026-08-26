# Vera Dockerfile

Dockerfile designed to make getting started with Vera easily.

Dockerfile can be used to join a pre-existing network or used for a standalone test envrionment.

To configure initialization set `MONIKER` for the node moniker and `CHAIN_ID` for the chain id.

Validator, comet p2p and comet validator keys can be recovered and loaded for previously created keys, otherwise new keys will be generated.

## Usage modes

### Validator Recovery mode
Validator recovery mode configures the vera node to recover a validator credentials

Set env var `MNEMONIC_PATH` to recover the vera validator key.
Set env var `CONSENSUS_KEY_PATH` to recover the CometBFT consensus key (ie. `priv_validator_key.json`).
Set env var `COMET_NODE_KEY_PATH` to recover the CometBFT p2p key (ie. `node_key.json`)
Set `GENESIS_PATH` to initialize the genesis file.

### RPC Mode
RPC Mode joins an existing network as an RPC Node with a new set of keys.

Set `GENESIS_PATH` to specify the network genesis.
Ensure `CHAIN_ID` matches the chain id in the genesis file.

### RPC with account recovery
To spin up an RPC node with a previously generated account key, follow the steps in RPC Mode and additionally set `MNEMONIC_PATH`.


## Standalone mode
Standalone mode is ideal for local experimentation and test environments.
During container startup, it generates a new network and genesis.

Set `STANDALONE=1` at time of container creation to force standalone mode, all recovery variables are ignored in standalone mode.

## Environment Variable Reference


- `MONIKER` sets the node moniker
- `CHAIN_ID` sets the id for the chain which will be initialized
- `GENESIS_PATH` is an optional variable which if set must point to a genesis file mounted in the container.
  The file is copied to the configuration directory during the first container initialization
  If empty, the entrypoint will generate a new genesis

- `MNEMONIC_PATH` is an optional varible which, if set, must point to a file containing a 
  cosmos key mnemonic. The mnemonic will be used to restore the node operator / validator key.
  If empty, the entrypoint will generate a new key

- `CONSENSUS_KEY_PATH` is an optional variable which, if set, must point to a file containg
  a comebft consesus key for the validator.
  If empty, the entrypoint will generate a new key

- `COMET_NODE_KEY_PATH` is an optional variable which, if set, must point to a file containg
  a comebft p2p node key.
  If empty, the entrypoint will generate a new key

- `COMET_CONFIG_PATH` is an optional variable which, if set, will overwrite
  the default cofig.toml with the provided file.

- `APP_CONFIG_PATH` is an optional variable which, if set, will overwrite
  the default app.toml with the provided file.

- `STANDALONE` if set to `1` will initialize a new Vera network / genesis for local usage.
  The network will with no fees, a single validator and a funded faucet account.