package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/sourcenetwork/sourcehub/app/params"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

// FaucetRequest represents a faucet request record.
type FaucetRequest struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
	TxHash  string `json:"tx_hash"`
}

// RegisterFaucetRoutes registers the faucet API routes.
func (app *App) RegisterFaucetRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	r := apiSvr.Router

	r.HandleFunc("/faucet/info", app.handleFaucetInfo(clientCtx)).Methods("GET")
	r.HandleFunc("/faucet/init-account", app.handleInitAccount(clientCtx)).Methods("POST")
	r.HandleFunc("/faucet/request", app.handleFaucetRequest(clientCtx)).Methods("POST")
}

// faucetEnabled returns true if the faucet is enabled, false otherwise.
func (app *App) faucetEnabled() bool {
	store := app.BaseApp.CommitMultiStore().GetKVStore(app.GetKey(authtypes.StoreKey))
	if store == nil {
		return false
	}
	bz := store.Get([]byte(appparams.EnableFaucetKey))
	return len(bz) > 0 && bz[0] == 0x01
}

// zeroFeeTxsAllowed returns true if zero fee transactions are allowed, false otherwise.
func (app *App) zeroFeeTxsAllowed() bool {
	store := app.BaseApp.CommitMultiStore().GetKVStore(app.GetKey(authtypes.StoreKey))
	if store == nil {
		return false
	}
	bz := store.Get([]byte(appparams.AllowZeroFeeTxsKey))
	return len(bz) > 0 && bz[0] == 0x01
}

// hasAddressRequested checks if an address has already requested funds.
func (app *App) hasAddressRequested(address string) bool {
	store := app.BaseApp.CommitMultiStore().GetKVStore(app.GetKey(params.FaucetStoreKey))
	if store == nil {
		return false
	}
	return store.Has([]byte(address))
}

// recordAddressRequested records that an address has requested funds.
func (app *App) recordAddressRequested(address, amount, txHash string) error {
	store := app.BaseApp.CommitMultiStore().GetKVStore(app.GetKey(params.FaucetStoreKey))
	if store == nil {
		return fmt.Errorf("faucet store not found")
	}

	request := FaucetRequest{
		Address: address,
		Amount:  amount,
		TxHash:  txHash,
	}

	bz, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal faucet request: %w", err)
	}

	store.Set([]byte(address), bz)
	return nil
}

// getRequestCount returns the number of addresses that have requested funds.
func (app *App) getRequestCount() int {
	store := app.BaseApp.CommitMultiStore().GetKVStore(app.GetKey(params.FaucetStoreKey))
	if store == nil {
		return 0
	}

	iterator := storetypes.KVStorePrefixIterator(store, []byte{})
	defer iterator.Close()

	count := 0
	for ; iterator.Valid(); iterator.Next() {
		count++
	}
	return count
}

// FaucetKeyConfig represents the structure of the faucet key JSON file.
type FaucetKeyConfig struct {
	Mnemonic string `json:"mnemonic"`
	Name     string `json:"name"`
	Address  string `json:"address"`
}

// getFaucetKey retrieves the faucet key from the faucet-key.json configuration file.
func (app *App) getFaucetKey() (keyring.Keyring, keyring.Record, error) {
	faucetKeyPath := filepath.Join(DefaultNodeHome, "config", "faucet-key.json")
	if _, err := os.Stat(faucetKeyPath); os.IsNotExist(err) {
		return nil, keyring.Record{}, fmt.Errorf("faucet key configuration file not found at %s", faucetKeyPath)
	}

	var faucetKeyConfig FaucetKeyConfig
	file, err := os.Open(faucetKeyPath)
	if err != nil {
		return nil, keyring.Record{}, fmt.Errorf("failed to open faucet key configuration file: %w", err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&faucetKeyConfig); err != nil {
		return nil, keyring.Record{}, fmt.Errorf("failed to decode faucet key configuration: %w", err)
	}

	kb := keyring.NewInMemory(app.appCodec)
	info, err := kb.NewAccount(faucetKeyConfig.Name, faucetKeyConfig.Mnemonic, "", "m/44'/118'/0'/0/0", hd.Secp256k1)
	if err != nil {
		return nil, keyring.Record{}, fmt.Errorf("failed to import faucet key: %w", err)
	}

	return kb, *info, nil
}

// handleFaucetRequest handles POST requests to request funds from the faucet.
func (app *App) handleFaucetRequest(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !app.faucetEnabled() {
			http.Error(w, "Faucet is not enabled", http.StatusServiceUnavailable)
			return
		}

		var req struct {
			Address string `json:"address"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		_, err := sdk.AccAddressFromBech32(req.Address)
		if err != nil {
			http.Error(w, "Invalid address", http.StatusBadRequest)
			return
		}

		// Check if address has already requested funds
		if app.hasAddressRequested(req.Address) {
			http.Error(w, "Address has already requested funds from the faucet", http.StatusForbidden)
			return
		}

		amount := "1000000000uopen" // 1000 open

		coins, err := sdk.ParseCoinsNormalized(amount)
		if err != nil {
			http.Error(w, "Invalid amount", http.StatusBadRequest)
			return
		}

		kb, faucetInfo, err := app.getFaucetKey()
		if err != nil {
			http.Error(w, fmt.Sprintf("Faucet not configured: %v", err), http.StatusInternalServerError)
			return
		}

		faucetAddress, err := faucetInfo.GetAddress()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get faucet address: %v", err), http.StatusInternalServerError)
			return
		}

		msg := banktypes.NewMsgSend(
			faucetAddress,
			sdk.MustAccAddressFromBech32(req.Address),
			coins,
		)

		txf := tx.Factory{}.
			WithTxConfig(clientCtx.TxConfig).
			WithAccountRetriever(clientCtx.AccountRetriever).
			WithChainID("sourcehub-dev").
			WithGas(200000).
			WithKeybase(kb)

		// Only add fees if zero fee transactions are not allowed
		if !app.zeroFeeTxsAllowed() {
			txf = txf.WithFees("200uopen")
		}

		faucetAccount, err := clientCtx.AccountRetriever.GetAccount(clientCtx, faucetAddress)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get faucet account: %v", err), http.StatusInternalServerError)
			return
		}

		txf = txf.WithAccountNumber(faucetAccount.GetAccountNumber()).
			WithSequence(faucetAccount.GetSequence())

		txn, err := txf.BuildUnsignedTx(msg)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to build transaction: %v", err), http.StatusInternalServerError)
			return
		}

		err = tx.Sign(r.Context(), txf, "faucet", txn, true)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to sign transaction: %v", err), http.StatusInternalServerError)
			return
		}

		txBytes, err := clientCtx.TxConfig.TxEncoder()(txn.GetTx())
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode transaction: %v", err), http.StatusInternalServerError)
			return
		}

		res, err := clientCtx.BroadcastTx(txBytes)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to broadcast transaction: %v", err), http.StatusInternalServerError)
			return
		}

		if res.Code == 0 {
			if err := app.recordAddressRequested(req.Address, amount, res.TxHash); err != nil {
				fmt.Printf("Failed to record faucet request: %v\n", err)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"txhash":  res.TxHash,
			"code":    res.Code,
			"raw_log": res.RawLog,
			"address": req.Address,
			"amount":  amount,
		})
	}
}

// handleFaucetInfo handles GET requests to get faucet information.
func (app *App) handleFaucetInfo(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !app.faucetEnabled() {
			http.Error(w, "Faucet is not enabled", http.StatusServiceUnavailable)
			return
		}

		_, faucetInfo, err := app.getFaucetKey()
		if err != nil {
			http.Error(w, fmt.Sprintf("Faucet not configured: %v", err), http.StatusNotFound)
			return
		}

		faucetAddress, err := faucetInfo.GetAddress()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get faucet address: %v", err), http.StatusInternalServerError)
			return
		}

		bankClient := banktypes.NewQueryClient(clientCtx)
		balance, err := bankClient.Balance(r.Context(), &banktypes.QueryBalanceRequest{
			Address: faucetAddress.String(),
			Denom:   "uopen",
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get faucet balance: %v", err), http.StatusInternalServerError)
			return
		}

		requestCount := app.getRequestCount()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"address":       faucetAddress.String(),
			"balance":       balance.Balance,
			"request_count": requestCount,
		})
	}
}

// handleInitAccount handles POST requests to initialize an account on chain.
// Accounts that are not yet registered in the auth module are initialized with 1 uopen.
func (app *App) handleInitAccount(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !app.faucetEnabled() {
			http.Error(w, "Faucet is not enabled", http.StatusServiceUnavailable)
			return
		}

		var req struct {
			Address string `json:"address"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		accAddr, err := sdk.AccAddressFromBech32(req.Address)
		if err != nil {
			http.Error(w, "Invalid address", http.StatusBadRequest)
			return
		}

		// Check if account already exists in the auth module
		existingAccount, err := clientCtx.AccountRetriever.GetAccount(clientCtx, accAddr)
		if err == nil && existingAccount != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"message": "Account already exists",
				"address": req.Address,
				"exists":  true,
			})
			return
		}

		kb, faucetInfo, err := app.getFaucetKey()
		if err != nil {
			http.Error(w, fmt.Sprintf("Faucet not configured: %v", err), http.StatusInternalServerError)
			return
		}

		faucetAddress, err := faucetInfo.GetAddress()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get faucet address: %v", err), http.StatusInternalServerError)
			return
		}

		// Create a bank send message with 1 uopen to initialize the account
		coins := sdk.NewCoins(sdk.NewInt64Coin("uopen", 1))
		msg := banktypes.NewMsgSend(
			faucetAddress,
			accAddr,
			coins,
		)

		txf := tx.Factory{}.
			WithTxConfig(clientCtx.TxConfig).
			WithAccountRetriever(clientCtx.AccountRetriever).
			WithChainID("sourcehub-dev").
			WithGas(200000).
			WithKeybase(kb)

		// Only add fees if zero fee transactions are not allowed
		if !app.zeroFeeTxsAllowed() {
			txf = txf.WithFees("200uopen")
		}

		faucetAccount, err := clientCtx.AccountRetriever.GetAccount(clientCtx, faucetAddress)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get faucet account: %v", err), http.StatusInternalServerError)
			return
		}

		txf = txf.WithAccountNumber(faucetAccount.GetAccountNumber()).
			WithSequence(faucetAccount.GetSequence())

		txn, err := txf.BuildUnsignedTx(msg)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to build transaction: %v", err), http.StatusInternalServerError)
			return
		}

		err = tx.Sign(r.Context(), txf, "faucet", txn, true)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to sign transaction: %v", err), http.StatusInternalServerError)
			return
		}

		txBytes, err := clientCtx.TxConfig.TxEncoder()(txn.GetTx())
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode transaction: %v", err), http.StatusInternalServerError)
			return
		}

		res, err := clientCtx.BroadcastTx(txBytes)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to broadcast transaction: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "Account initialized successfully",
			"txhash":  res.TxHash,
			"code":    res.Code,
			"raw_log": res.RawLog,
			"address": req.Address,
			"amount":  "1uopen",
			"exists":  false,
		})
	}
}
