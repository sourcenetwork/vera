package tier

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/store"
	"cosmossdk.io/depinject"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	appparams "github.com/sourcenetwork/sourcehub/app/params"

	// this line is used by starport scaffolding # 1

	modulev1beta1 "github.com/sourcenetwork/sourcehub/api/sourcehub/tier/module/v1beta1"

	epochstypes "github.com/sourcenetwork/sourcehub/x/epochs/types"

	"github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

var (
	_ module.AppModuleBasic      = (*AppModule)(nil)
	_ module.AppModuleSimulation = (*AppModule)(nil)
	_ module.HasGenesis          = (*AppModule)(nil)
	_ module.HasInvariants       = (*AppModule)(nil)
	_ module.HasConsensusVersion = (*AppModule)(nil)

	_ appmodule.AppModule       = (*AppModule)(nil)
	_ appmodule.HasBeginBlocker = (*AppModule)(nil)
	_ appmodule.HasEndBlocker   = (*AppModule)(nil)
)

// ----------------------------------------------------------------------------
// AppModuleBasic
// ----------------------------------------------------------------------------

// AppModuleBasic implements the AppModuleBasic interface that defines the
// independent methods a Cosmos SDK module needs to implement.
type AppModuleBasic struct {
	cdc codec.BinaryCodec
}

func NewAppModuleBasic(cdc codec.BinaryCodec) AppModuleBasic {
	return AppModuleBasic{cdc: cdc}
}

// Name returns the name of the module as a string.
func (AppModuleBasic) Name() string {
	return types.ModuleName
}

// RegisterLegacyAminoCodec registers the amino codec for the module, which is used
// to marshal and unmarshal structs to/from []byte in order to persist them in the module's KVStore.
func (AppModuleBasic) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {}

// RegisterInterfaces registers a module's interface types and their concrete implementations as proto.Message.
func (a AppModuleBasic) RegisterInterfaces(reg cdctypes.InterfaceRegistry) {
	types.RegisterInterfaces(reg)
}

// DefaultGenesis returns a default GenesisState for the module, marshalled to json.RawMessage.
// The default GenesisState need to be defined by the module developer and is primarily used for testing.
func (AppModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	return cdc.MustMarshalJSON(types.DefaultGenesis())
}

// ValidateGenesis used to validate the GenesisState, given in its json.RawMessage form.
func (AppModuleBasic) ValidateGenesis(cdc codec.JSONCodec, config client.TxEncodingConfig, bz json.RawMessage) error {
	var genState types.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genState); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", types.ModuleName, err)
	}
	return genState.Validate()
}

// RegisterGRPCGatewayRoutes registers the gRPC Gateway routes for the module.
func (AppModuleBasic) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	if err := types.RegisterQueryHandlerClient(context.Background(), mux, types.NewQueryClient(clientCtx)); err != nil {
		panic(err)
	}
}

// ----------------------------------------------------------------------------
// AppModule
// ----------------------------------------------------------------------------

// AppModule implements the AppModule interface that defines the inter-dependent methods that modules need to implement
type AppModule struct {
	AppModuleBasic

	keeper     keeper.Keeper
	bankKeeper types.BankKeeper
}

func NewAppModule(
	cdc codec.Codec,
	keeper keeper.Keeper,
	bankKeeper types.BankKeeper,
) AppModule {
	return AppModule{
		AppModuleBasic: NewAppModuleBasic(cdc),
		keeper:         keeper,
		bankKeeper:     bankKeeper,
	}
}

// RegisterServices registers a gRPC query service to respond to the module-specific gRPC queries
func (am AppModule) RegisterServices(cfg module.Configurator) {
	types.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	types.RegisterQueryServer(cfg.QueryServer(), keeper.NewQuerier(am.keeper))
}

// RegisterInvariants registers the invariants of the module. If an invariant deviates from its predicted value, the InvariantRegistry triggers appropriate logic (most often the chain will be halted)
func (am AppModule) RegisterInvariants(_ sdk.InvariantRegistry) {}

// InitGenesis performs the module's genesis initialization. It returns no validator updates.
func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, gs json.RawMessage) {
	var genState types.GenesisState
	// Initialize global index to index in genesis state
	cdc.MustUnmarshalJSON(gs, &genState)

	InitGenesis(ctx, am.keeper, genState)
}

// ExportGenesis returns the module's exported genesis state as raw JSON bytes.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	genState := ExportGenesis(ctx, am.keeper)
	return cdc.MustMarshalJSON(genState)
}

// ConsensusVersion is a sequence number for state-breaking change of the module.
// It should be incremented on each consensus-breaking change introduced by the module.
// To avoid wrong/empty versions, the initial version should be set to 1.
func (AppModule) ConsensusVersion() uint64 { return 1 }

// BeginBlock contains the logic that is automatically triggered at the beginning of each block.
// The begin block implementation is optional.
func (am AppModule) BeginBlock(ctx context.Context) error {
	params := am.keeper.GetParams(ctx)

	// Process rewards every N blocks
	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if height%params.ProcessRewardsInterval != 0 {
		return nil
	}

	tierModuleAddr := authtypes.NewModuleAddress(types.ModuleName)
	err := am.keeper.GetStakingKeeper().IterateDelegations(ctx, tierModuleAddr, func(index int64, delegation stakingtypes.DelegationI) bool {
		// Claim rewards for the tier module from this validator
		valAddr := types.MustValAddressFromBech32(delegation.GetValidatorAddr())
		rewards, err := am.keeper.GetDistributionKeeper().WithdrawDelegationRewards(ctx, tierModuleAddr, valAddr)
		if err != nil {
			am.keeper.Logger().Error("Failed to claim tier module staking rewards", "error", err)
			return false
		}

		// Proceed to the next record if there are no rewards
		if rewards.IsZero() {
			am.keeper.Logger().Info("No tier module staking rewards in validator", "validator", valAddr)
			return false
		}

		totalAmount := rewards.AmountOf(appparams.DefaultBondDenom)
		amountToDevPool := totalAmount.MulRaw(params.DeveloperPoolFee).QuoRaw(100)
		amountToInsurancePool := totalAmount.MulRaw(params.InsurancePoolFee).QuoRaw(100)
		amountToBurn := totalAmount.Sub(amountToDevPool).Sub(amountToInsurancePool)

		// Send InsurancePoolFee to the insurance pool if threshold not reached, update amountToDevPool otherwise
		if !amountToInsurancePool.IsZero() {
			insurancePoolAddr := authtypes.NewModuleAddress(types.InsurancePoolName)
			insurancePoolBalance := am.keeper.GetBankKeeper().GetBalance(ctx, insurancePoolAddr, appparams.DefaultBondDenom)
			if insurancePoolBalance.Amount.LT(math.NewInt(params.InsurancePoolThreshold)) {
				insuranceCoins := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, amountToInsurancePool))
				err := am.keeper.GetBankKeeper().SendCoinsFromModuleToModule(ctx, types.ModuleName, types.InsurancePoolName, insuranceCoins)
				if err != nil {
					am.keeper.Logger().Error("Failed to send rewards to the insurance pool", "error", err)
					return false
				}
			} else {
				amountToDevPool = amountToDevPool.Add(amountToInsurancePool)
			}
		}

		// Send DeveloperPoolFee to the developer pool
		if !amountToDevPool.IsZero() {
			devPoolCoins := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, amountToDevPool))
			err := am.keeper.GetBankKeeper().SendCoinsFromModuleToModule(ctx, types.ModuleName, types.DeveloperPoolName, devPoolCoins)
			if err != nil {
				am.keeper.Logger().Error("Failed to send rewards to the developer pool", "error", err)
				return false
			}
		}

		// Burn remaining tier module staking rewards
		if !amountToBurn.IsZero() {
			burnCoins := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, amountToBurn))
			err := am.keeper.GetBankKeeper().BurnCoins(ctx, types.ModuleName, burnCoins)
			if err != nil {
				am.keeper.Logger().Error("Failed to burn tier module staking rewards", "error", err)
				return false
			}
		}

		return false
	})

	if err != nil {
		am.keeper.Logger().Error("Error iterating over tier module delegations", "error", err)
		return err
	}

	return nil
}

// EndBlock contains the logic that is automatically triggered at the end of each block.
// The end block implementation is optional.
func (am AppModule) EndBlock(_ context.Context) error {
	return nil
}

// IsOnePerModuleType implements the depinject.OnePerModuleType interface.
func (am AppModule) IsOnePerModuleType() {}

// IsAppModule implements the appmodule.AppModule interface.
func (am AppModule) IsAppModule() {}

// ----------------------------------------------------------------------------
// App Wiring Setup
// ----------------------------------------------------------------------------

func init() {
	appmodule.Register(
		&modulev1beta1.Module{},
		appmodule.Provide(ProvideModule),
	)
}

type ModuleInputs struct {
	depinject.In

	StoreService store.KVStoreService
	Cdc          codec.Codec
	Config       *modulev1beta1.Module
	Logger       log.Logger

	BankKeeper         types.BankKeeper
	StakingKeeper      types.StakingKeeper
	EpochsKeeper       types.EpochsKeeper
	DistributionKeeper types.DistributionKeeper
}

type ModuleOutputs struct {
	depinject.Out

	TierKeeper keeper.Keeper
	Module     appmodule.AppModule
	Hooks      epochstypes.EpochsHooksWrapper
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	// default to governance authority if not provided
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	if in.Config.Authority != "" {
		authority = authtypes.NewModuleAddressOrBech32Address(in.Config.Authority)
	}
	k := keeper.NewKeeper(
		in.Cdc,
		in.StoreService,
		in.Logger,
		authority.String(),
		in.BankKeeper,
		in.StakingKeeper,
		in.EpochsKeeper,
		in.DistributionKeeper,
	)

	m := NewAppModule(
		in.Cdc,
		k,
		in.BankKeeper,
	)

	return ModuleOutputs{
		TierKeeper: k,
		Module:     m,
		Hooks:      epochstypes.EpochsHooksWrapper{EpochHooks: k.EpochHooks()},
	}
}
