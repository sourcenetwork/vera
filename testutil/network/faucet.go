package network

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	"github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/sourcenetwork/sourcehub/app"
	"github.com/sourcenetwork/sourcehub/app/params"
)

const (
	faucetAddress    = "source12d9hjf0639k995venpv675sju9ltsvf8u5c9jt"
	faucetBalance    = 100000000000000 // 100m open
	faucetKeyContent = `{
  "mnemonic": "comic very pond victory suit tube ginger antique life then core warm loyal deliver iron fashion erupt husband weekend monster sunny artist empty uphold",
  "name": "faucet",
  "address": "source12d9hjf0639k995venpv675sju9ltsvf8u5c9jt"
}`
	defaultDirPerm  = 0755
	defaultFilePerm = 0644
	faucetKeyFile   = "faucet-key.json"
	configDir       = "config"
)

// setupFaucetKeyFiles creates the faucet key files in the required locations.
func setupFaucetKeyFiles(val network.ValidatorI) error {
	faucetKeyPath := filepath.Join(val.GetCtx().Config.RootDir, configDir, faucetKeyFile)
	if err := os.MkdirAll(filepath.Dir(faucetKeyPath), defaultDirPerm); err != nil {
		return fmt.Errorf("failed to create faucet key directory: %w", err)
	}
	if err := os.WriteFile(faucetKeyPath, []byte(faucetKeyContent), defaultFilePerm); err != nil {
		return fmt.Errorf("failed to write faucet key file: %w", err)
	}
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home directory: %w", err)
	}
	defaultNodeHome := filepath.Join(userHomeDir, ".sourcehub")
	defaultFaucetKeyPath := filepath.Join(defaultNodeHome, configDir, faucetKeyFile)
	if err := os.MkdirAll(filepath.Dir(defaultFaucetKeyPath), defaultDirPerm); err != nil {
		return fmt.Errorf("failed to create default faucet key directory: %w", err)
	}
	if err := os.WriteFile(defaultFaucetKeyPath, []byte(faucetKeyContent), defaultFilePerm); err != nil {
		return fmt.Errorf("failed to write default faucet key file: %w", err)
	}
	return nil
}

// enableFaucetInGenesis enables the faucet in the genesis state.
func enableFaucetInGenesis(cfg *network.Config) error {
	appParamsGenesis := params.AppParamsGenesis{
		AllowZeroFeeTxs: true,
		EnableFaucet:    true,
	}
	appParamsBytes, err := json.Marshal(&appParamsGenesis)
	if err != nil {
		return fmt.Errorf("could not marshal app_params: %w", err)
	}
	cfg.GenesisState[params.AppParamsGenesisKey] = appParamsBytes
	if err := addFaucetAccountToGenesis(cfg); err != nil {
		return fmt.Errorf("failed to add faucet account to genesis: %w", err)
	}
	if err := addFaucetBalanceToGenesis(cfg); err != nil {
		return fmt.Errorf("failed to add faucet balance to genesis: %w", err)
	}
	return nil
}

// addFaucetAccountToGenesis adds the faucet account to the auth genesis state.
func addFaucetAccountToGenesis(cfg *network.Config) error {
	var authGenState authtypes.GenesisState
	if authGenStateBytes, exists := cfg.GenesisState[authtypes.ModuleName]; exists {
		if err := json.Unmarshal(authGenStateBytes, &authGenState); err != nil {
			authGenState = *authtypes.DefaultGenesisState()
		}
	} else {
		authGenState = *authtypes.DefaultGenesisState()
	}
	faucetAccAddr, err := sdk.AccAddressFromBech32(faucetAddress)
	if err != nil {
		return fmt.Errorf("invalid faucet address: %w", err)
	}
	faucetAccount := authtypes.NewBaseAccount(faucetAccAddr, nil, 0, 0)
	accounts, err := authtypes.PackAccounts([]authtypes.GenesisAccount{faucetAccount})
	if err != nil {
		return fmt.Errorf("failed to pack accounts: %w", err)
	}
	authGenState.Accounts = append(authGenState.Accounts, accounts...)
	tempApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, sims.EmptyAppOptions{})
	if err != nil {
		return fmt.Errorf("failed to create temp app: %w", err)
	}
	appCodec := tempApp.AppCodec()
	cfg.GenesisState[authtypes.ModuleName] = appCodec.MustMarshalJSON(&authGenState)
	return nil
}

// addFaucetBalanceToGenesis adds the faucet balance to the bank genesis state.
func addFaucetBalanceToGenesis(cfg *network.Config) error {
	var bankGenState banktypes.GenesisState
	if bankGenStateBytes, exists := cfg.GenesisState[banktypes.ModuleName]; exists {
		if err := json.Unmarshal(bankGenStateBytes, &bankGenState); err != nil {
			bankGenState = *banktypes.DefaultGenesisState()
		}
	} else {
		bankGenState = *banktypes.DefaultGenesisState()
	}
	faucetCoins := sdk.NewCoins(sdk.NewCoin(params.DefaultBondDenom, math.NewInt(faucetBalance)))
	bankGenState.Balances = append(bankGenState.Balances, banktypes.Balance{
		Address: faucetAddress,
		Coins:   faucetCoins,
	})
	tempApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, sims.EmptyAppOptions{})
	if err != nil {
		return fmt.Errorf("failed to create temp app: %w", err)
	}
	appCodec := tempApp.AppCodec()
	cfg.GenesisState[banktypes.ModuleName] = appCodec.MustMarshalJSON(&bankGenState)
	return nil
}
