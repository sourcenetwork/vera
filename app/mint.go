package app

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/log"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	tierkeeper "github.com/sourcenetwork/sourcehub/x/tier/keeper"
)

var _ appmodule.AppModule = CustomMintModule{}

// CustomMintModule overrides default mint module.
type CustomMintModule struct {
	*mint.AppModule
	mintKeeper mintkeeper.Keeper
	tierKeeper tierkeeper.Keeper
	cdc        codec.Codec
}

// NewCustomMintModule creates new custom mint module instance.
func NewCustomMintModule(
	oldMintModule mint.AppModule,
	mintKeeper mintkeeper.Keeper,
	tierKeeper tierkeeper.Keeper,
	cdc codec.Codec,
) CustomMintModule {
	return CustomMintModule{
		AppModule:  &oldMintModule,
		mintKeeper: mintKeeper,
		tierKeeper: tierKeeper,
		cdc:        cdc,
	}
}

// CustomMintQueryServer overrides default mint query server.
type CustomMintQueryServer struct {
	minttypes.QueryServer
	mintKeeper mintkeeper.Keeper
	tierKeeper tierkeeper.Keeper
	logger     log.Logger
}

// NewCustomMintQueryServer creates new custom mint query server instance.
func NewCustomMintQueryServer(
	defaultQueryServer minttypes.QueryServer,
	mintKeeper mintkeeper.Keeper,
	tierKeeper tierkeeper.Keeper,
	logger log.Logger,
) minttypes.QueryServer {
	return &CustomMintQueryServer{
		QueryServer: defaultQueryServer,
		mintKeeper:  mintKeeper,
		tierKeeper:  tierKeeper,
		logger:      logger,
	}
}

func getDelegatorStakeRatio(ctx context.Context, k tierkeeper.Keeper) (math.LegacyDec, error) {
	// Get the total bonded stake
	totalStake, err := k.GetStakingKeeper().TotalBondedTokens(ctx)
	if err != nil {
		return math.LegacyOneDec(), err
	}

	// Get developer stake and fees
	devStake := k.GetTotalLockupsAmount(ctx)
	params := k.GetParams(ctx)
	totalFees := params.DeveloperPoolFee + params.InsurancePoolFee
	devStakeMinusFees := devStake.MulRaw(100 - totalFees).QuoRaw(100)

	// Calculate the delegator stake
	delStake, err := totalStake.SafeSub(devStakeMinusFees)
	if err != nil {
		return math.LegacyOneDec(), err
	}

	if !totalStake.IsPositive() || !delStake.IsPositive() {
		return math.LegacyOneDec(), fmt.Errorf("non-positive totalStake/delStake")
	}

	// Calculate the delegator stake ratio
	delStakeRatio := delStake.ToLegacyDec().Quo(totalStake.ToLegacyDec())

	return delStakeRatio, nil
}

// Inflation overrides the default mint module inflation query.
func (q CustomMintQueryServer) Inflation(ctx context.Context, _ *minttypes.QueryInflationRequest) (*minttypes.QueryInflationResponse, error) {
	// Fetch the minter state
	minter, err := q.mintKeeper.Minter.Get(ctx)
	if err != nil {
		return nil, err
	}

	// Get the delegator stake ratio
	delStakeRatio, err := getDelegatorStakeRatio(ctx, q.tierKeeper)
	if err != nil {
		q.logger.Info("Returning default inflation", "inflation", minter.Inflation)
		return &minttypes.QueryInflationResponse{Inflation: minter.Inflation}, nil
	}

	// Calculate the effective inflation
	effectiveInflation := minter.Inflation.Mul(delStakeRatio)
	q.logger.Info("Returning effective inflation", "inflation", effectiveInflation)

	return &minttypes.QueryInflationResponse{Inflation: effectiveInflation}, nil
}

// RegisterServices registers default message server and custom query server.
func (cm CustomMintModule) RegisterServices(cfg module.Configurator) {
	minttypes.RegisterMsgServer(cfg.MsgServer(), mintkeeper.NewMsgServerImpl(cm.mintKeeper))
	defaultMintQueryServer := mintkeeper.NewQueryServerImpl(cm.mintKeeper)
	customMintQueryServer := NewCustomMintQueryServer(defaultMintQueryServer, cm.mintKeeper, cm.tierKeeper, cm.tierKeeper.Logger())
	minttypes.RegisterQueryServer(cfg.QueryServer(), customMintQueryServer)
}

// InitGenesis initializes default state for the mint keeper and overrides the default params.
func (cm CustomMintModule) InitGenesis(ctx context.Context, cdc codec.JSONCodec, data json.RawMessage) {
	genesisState := minttypes.DefaultGenesisState()

	genesisState.Minter = minttypes.DefaultInitialMinter()
	genesisState.Minter.Inflation = math.LegacyMustNewDecFromStr(appparams.InitialInflation)
	genesisState.Params = minttypes.DefaultParams()
	genesisState.Params.MintDenom = appparams.DefaultBondDenom
	genesisState.Params.BlocksPerYear = appparams.BlocksPerYear
	genesisState.Params.InflationMin = math.LegacyMustNewDecFromStr(appparams.InflationMin)
	genesisState.Params.InflationMax = math.LegacyMustNewDecFromStr(appparams.InflationMax)
	genesisState.Params.InflationRateChange = math.LegacyMustNewDecFromStr(appparams.InflationRateChange)

	if err := cm.mintKeeper.Minter.Set(ctx, genesisState.Minter); err != nil {
		panic(err)
	}

	if err := cm.mintKeeper.Params.Set(ctx, genesisState.Params); err != nil {
		panic(err)
	}
}

// registerCustomMintModule registers the custom mint module.
func (app *App) registerCustomMintModule() CustomMintModule {
	mintStoreKey := storetypes.NewKVStoreKey(minttypes.StoreKey)
	if err := app.RegisterStores(mintStoreKey); err != nil {
		panic(err)
	}

	app.MintKeeper = mintkeeper.NewKeeper(
		app.appCodec,
		runtime.NewKVStoreService(app.GetKey(minttypes.StoreKey)),
		app.StakingKeeper,
		app.AccountKeeper,
		app.BankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	defaultMintModule := mint.NewAppModule(app.appCodec, app.MintKeeper, app.AccountKeeper, nil, nil)
	customMintModule := NewCustomMintModule(defaultMintModule, app.MintKeeper, app.TierKeeper, app.appCodec)
	if err := app.RegisterModules(customMintModule); err != nil {
		panic(err)
	}

	return customMintModule
}

// RegisterMintInterfaces registers interfaces for the mint module and returns mint.AppModule.
func RegisterMintInterfaces(registry codectypes.InterfaceRegistry) appmodule.AppModule {
	module := mint.AppModule{}
	module.RegisterInterfaces(registry)
	return module
}
