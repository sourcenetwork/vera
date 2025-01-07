package types

import (
	"context"
	time "time"

	"cosmossdk.io/math"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	epochstypes "github.com/sourcenetwork/sourcehub/x/epochs/types"
)

// EpochsKeeper defines the expected interface for the Epochs module.
type EpochsKeeper interface {
	GetEpochInfo(ctx context.Context, identifier string) epochstypes.EpochInfo
	SetEpochInfo(ctx context.Context, info epochstypes.EpochInfo)
}

// StakingKeeper defines the expected interface for the Staking module.
type StakingKeeper interface {
	Delegate(ctx context.Context, delAddr sdk.AccAddress, bondAmt math.Int, tokenSrc stakingtypes.BondStatus,
		validator stakingtypes.Validator, subtractAccount bool) (newShares math.LegacyDec, err error)
	Undelegate(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, sharesAmount math.LegacyDec) (
		time.Time, math.Int, error)
	BeginRedelegation(ctx context.Context, delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress,
		sharesAmount math.LegacyDec) (completionTime time.Time, err error)
	BondDenom(ctx context.Context) (string, error)
	GetValidator(ctx context.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, err error)
	IterateValidators(context.Context, func(index int64, validator stakingtypes.ValidatorI) (stop bool)) error
	TotalBondedTokens(context.Context) (math.Int, error)
	ValidateUnbondAmount(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) (
		shares math.LegacyDec, err error)
	GetUnbondingDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (
		ubd stakingtypes.UnbondingDelegation, err error)
	SetUnbondingDelegation(ctx context.Context, ubd stakingtypes.UnbondingDelegation) error
	RemoveUnbondingDelegation(ctx context.Context, ubd stakingtypes.UnbondingDelegation) error
	SetValidatorByConsAddr(ctx context.Context, addr stakingtypes.Validator) error
	SetValidator(ctx context.Context, addr stakingtypes.Validator) error
	CompleteUnbonding(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (sdk.Coins, error)
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error

	DelegateCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	UndelegateCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error

	// ViewKeeper interface
	IterateAllBalances(ctx context.Context, cb func(addr sdk.AccAddress, coin sdk.Coin) bool)
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// DistributionKeeper defines the expected interface for the distribution module.
type DistributionKeeper interface {
	AllocateTokensToValidator(ctx context.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins) error
	AllocateTokens(ctx context.Context, totalReward int64, bondedValidators []abcitypes.VoteInfo) error
	GetValidatorOutstandingRewards(ctx context.Context, valAddr sdk.ValAddress) (distrtypes.ValidatorOutstandingRewards, error)
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}
