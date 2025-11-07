package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/sourcenetwork/sourcehub/x/feegrant"

	faucettypes "github.com/sourcenetwork/sourcehub/app/faucet/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

// Faucet constants
const (
	FaucetRequestAmount            = 1000000000  // 1,000 $OPEN
	DefaultAllowanceAmount         = 10000000000 // 10,000 $OPEN
	DefaultAllowanceExpirationDays = 30
)

// getFaucetConfig extracts faucet configuration from app options.
func getFaucetConfig(appOpts servertypes.AppOptions) appparams.FaucetConfig {
	var faucetConfig appparams.FaucetConfig

	if enableFaucet := appOpts.Get("faucet.enable_faucet"); enableFaucet != nil {
		if boolVal, ok := enableFaucet.(bool); ok {
			faucetConfig.EnableFaucet = boolVal
		}
	}

	return faucetConfig
}

// shouldRegisterFaucetRoutes determines if faucet routes should be registered based on configuration.
func shouldRegisterFaucetRoutes(appOpts servertypes.AppOptions) bool {
	faucetConfig := getFaucetConfig(appOpts)
	return faucetConfig.EnableFaucet
}

// RegisterFaucetRoutes registers the faucet API routes.
func (app *App) RegisterFaucetRoutes(apiSvr *api.Server, apiConfig config.APIConfig, appOpts servertypes.AppOptions) {
	if !shouldRegisterFaucetRoutes(appOpts) {
		return
	}

	clientCtx := apiSvr.ClientCtx
	rtr := apiSvr.Router

	rtr.HandleFunc("/faucet/info", app.handleFaucetInfo(clientCtx)).Methods("GET")
	rtr.HandleFunc("/faucet/init-account", app.handleInitAccount(clientCtx)).Methods("POST")
	rtr.HandleFunc("/faucet/request", app.handleFaucetRequest(clientCtx)).Methods("POST")
	rtr.HandleFunc("/faucet/grant-allowance", app.handleGrantAllowance(clientCtx)).Methods("POST")
	rtr.HandleFunc("/faucet/grant-did-allowance", app.handleGrantDIDAllowance(clientCtx)).Methods("POST")
}

// zeroFeeTxsAllowed returns true if zero fee transactions are allowed, false otherwise.
func (app *App) zeroFeeTxsAllowed() bool {
	ctx := sdk.NewContext(app.CommitMultiStore(), cmtproto.Header{}, false, app.Logger())
	return app.HubKeeper != nil && app.HubKeeper.GetChainConfig(ctx).AllowZeroFeeTxs
}

// hasAddressRequested checks if an address has already requested funds.
func (app *App) hasAddressRequested(address string) bool {
	store := app.BaseApp.CommitMultiStore().GetKVStore(app.GetKey(appparams.FaucetStoreKey))
	if store == nil {
		return false
	}
	return store.Has([]byte(address))
}

// recordAddressRequested records that an address has requested funds.
func (app *App) recordAddressRequested(address string, amount sdk.Coins, txHash string) error {
	store := app.BaseApp.CommitMultiStore().GetKVStore(app.GetKey(appparams.FaucetStoreKey))
	if store == nil {
		return fmt.Errorf("faucet store not found")
	}

	request := &faucettypes.FaucetRequestRecord{
		Address: address,
		Amount:  amount[0],
		TxHash:  txHash,
	}

	bz, err := proto.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal faucet request: %w", err)
	}

	store.Set([]byte(address), bz)
	return nil
}

// getRequestCount returns the number of addresses that have requested funds.
func (app *App) getRequestCount() int {
	store := app.BaseApp.CommitMultiStore().GetKVStore(app.GetKey(appparams.FaucetStoreKey))
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
		var req faucettypes.FaucetRequest

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

		coins := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroOpenDenom, FaucetRequestAmount))

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

		chainID := clientCtx.ChainID
		if chainID == "" {
			chainID = "sourcehub-dev"
		}
		txf := tx.Factory{}.
			WithTxConfig(clientCtx.TxConfig).
			WithAccountRetriever(clientCtx.AccountRetriever).
			WithChainID(chainID).
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

		res, err := clientCtx.BroadcastTxSync(txBytes)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to broadcast transaction: %v", err), http.StatusInternalServerError)
			return
		}

		if res.Code == 0 {
			if err := app.recordAddressRequested(req.Address, coins, res.TxHash); err != nil {
				fmt.Printf("Failed to record faucet request: %v\n", err)
			}
		}

		response := &faucettypes.FaucetResponse{
			Txhash:  res.TxHash,
			Code:    res.Code,
			RawLog:  res.RawLog,
			Address: req.Address,
			Amount:  sdk.NewInt64Coin(appparams.MicroOpenDenom, FaucetRequestAmount),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// handleFaucetInfo handles GET requests to get faucet information.
func (app *App) handleFaucetInfo(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			Denom:   appparams.MicroOpenDenom,
		})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get faucet balance: %v", err), http.StatusInternalServerError)
			return
		}

		requestCount := app.getRequestCount()

		response := &faucettypes.FaucetInfoResponse{
			Address:      faucetAddress.String(),
			Balance:      *balance.Balance,
			RequestCount: int32(requestCount),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// handleInitAccount handles POST requests to initialize an account on chain.
// Accounts that are not yet registered in the auth module are initialized with 1 uopen.
func (app *App) handleInitAccount(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req faucettypes.InitAccountRequest

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
			response := &faucettypes.InitAccountResponse{
				Message: "Account already exists",
				Address: req.Address,
				Exists:  true,
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
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
		coins := sdk.NewCoins(sdk.NewInt64Coin(appparams.MicroOpenDenom, 1))
		msg := banktypes.NewMsgSend(
			faucetAddress,
			accAddr,
			coins,
		)

		chainID := clientCtx.ChainID
		if chainID == "" {
			chainID = "sourcehub-dev"
		}
		txf := tx.Factory{}.
			WithTxConfig(clientCtx.TxConfig).
			WithAccountRetriever(clientCtx.AccountRetriever).
			WithChainID(chainID).
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

		res, err := clientCtx.BroadcastTxSync(txBytes)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to broadcast transaction: %v", err), http.StatusInternalServerError)
			return
		}

		response := &faucettypes.InitAccountResponse{
			Message: "Account initialized successfully",
			Txhash:  res.TxHash,
			Code:    res.Code,
			RawLog:  res.RawLog,
			Address: req.Address,
			Amount:  sdk.NewInt64Coin(appparams.MicroOpenDenom, 1),
			Exists:  false,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// handleGrantAllowance handles POST requests to grant fee allowances from the faucet.
func (app *App) handleGrantAllowance(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req faucettypes.GrantAllowanceRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		granteeAddr, err := sdk.AccAddressFromBech32(req.Address)
		if err != nil {
			http.Error(w, "Invalid address", http.StatusBadRequest)
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

		amountLimit := sdk.NewInt64Coin(appparams.MicroOpenDenom, DefaultAllowanceAmount)
		if !req.AmountLimit.Amount.IsNil() && req.AmountLimit.Amount.IsPositive() {
			amountLimit = req.AmountLimit
		}

		expiration := time.Now().AddDate(0, 0, DefaultAllowanceExpirationDays)
		if req.Expiration != nil {
			expiration = *req.Expiration
		}

		// Create basic allowance
		spendLimit := sdk.NewCoins(amountLimit)
		allowance := &feegrant.BasicAllowance{
			SpendLimit: spendLimit,
			Expiration: &expiration,
		}

		msg, err := feegrant.NewMsgGrantAllowance(allowance, faucetAddress, granteeAddr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create grant allowance message: %v", err), http.StatusInternalServerError)
			return
		}

		chainID := clientCtx.ChainID
		if chainID == "" {
			chainID = "sourcehub-dev"
		}
		txf := tx.Factory{}.
			WithTxConfig(clientCtx.TxConfig).
			WithAccountRetriever(clientCtx.AccountRetriever).
			WithChainID(chainID).
			WithGas(300000).
			WithKeybase(kb)

		// Only add fees if zero fee transactions are not allowed
		if !app.zeroFeeTxsAllowed() {
			txf = txf.WithFees("300uopen")
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

		res, err := clientCtx.BroadcastTxSync(txBytes)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to broadcast transaction: %v", err), http.StatusInternalServerError)
			return
		}

		response := &faucettypes.GrantAllowanceResponse{
			Message:     "Fee allowance granted successfully",
			Txhash:      res.TxHash,
			Code:        res.Code,
			RawLog:      res.RawLog,
			Granter:     faucetAddress.String(),
			Grantee:     req.Address,
			AmountLimit: amountLimit,
			Expiration:  &expiration,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// handleGrantDIDAllowance handles POST requests to grant fee allowances to a DID from the faucet.
func (app *App) handleGrantDIDAllowance(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req faucettypes.GrantDIDAllowanceRequest

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Did == "" || len(req.Did) < 4 || req.Did[:4] != "did:" {
			http.Error(w, "Invalid DID", http.StatusBadRequest)
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

		amountLimit := sdk.NewInt64Coin(appparams.MicroOpenDenom, DefaultAllowanceAmount)
		if !req.AmountLimit.Amount.IsNil() && req.AmountLimit.Amount.IsPositive() {
			amountLimit = req.AmountLimit
		}

		expiration := time.Now().AddDate(0, 0, DefaultAllowanceExpirationDays)
		if req.Expiration != nil {
			expiration = *req.Expiration
		}

		// Create basic allowance
		spendLimit := sdk.NewCoins(amountLimit)
		allowance := &feegrant.BasicAllowance{
			SpendLimit: spendLimit,
			Expiration: &expiration,
		}

		msg, err := feegrant.NewMsgGrantDIDAllowance(allowance, faucetAddress, req.Did)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to create grant DID allowance message: %v", err), http.StatusInternalServerError)
			return
		}

		chainID := clientCtx.ChainID
		if chainID == "" {
			chainID = "sourcehub-dev"
		}
		txf := tx.Factory{}.
			WithTxConfig(clientCtx.TxConfig).
			WithAccountRetriever(clientCtx.AccountRetriever).
			WithChainID(chainID).
			WithGas(300000).
			WithKeybase(kb)

		// Only add fees if zero fee transactions are not allowed
		if !app.zeroFeeTxsAllowed() {
			txf = txf.WithFees("300uopen")
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

		res, err := clientCtx.BroadcastTxSync(txBytes)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to broadcast transaction: %v", err), http.StatusInternalServerError)
			return
		}

		response := &faucettypes.GrantDIDAllowanceResponse{
			Message:     "DID fee allowance granted successfully",
			Txhash:      res.TxHash,
			Code:        res.Code,
			RawLog:      res.RawLog,
			Granter:     faucetAddress.String(),
			GranteeDid:  req.Did,
			AmountLimit: amountLimit,
			Expiration:  &expiration,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
