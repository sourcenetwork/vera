package keeper_test

import (
	"testing"
	"time"

	cosmosed25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/sourcenetwork/sourcehub/app"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	testutil "github.com/sourcenetwork/sourcehub/testutil"
	"github.com/sourcenetwork/sourcehub/testutil/sample"
	tierkeeper "github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
	"github.com/stretchr/testify/require"
)

func init() {
	app.SetConfig(false)
}

// initializeValidator creates a validator and verifies that it was set correctly.
func initializeValidator(t *testing.T, k *keeper.Keeper, ctx sdk.Context, valAddr sdk.ValAddress, initialTokens math.Int) {
	validator := testutil.CreateTestValidator(t, ctx, k, valAddr, cosmosed25519.GenPrivKey().PubKey(), initialTokens)
	gotValidator, err := k.GetValidator(ctx, valAddr)
	require.Nil(t, err)
	require.Equal(t, validator.OperatorAddress, gotValidator.OperatorAddress)
}

// initializeDelegator initializes ba delegator with balance.
func initializeDelegator(t *testing.T, k *tierkeeper.Keeper, ctx sdk.Context, delAddr sdk.AccAddress, initialBalance math.Int) {
	initialDelegatorBalance := sdk.NewCoins(sdk.NewCoin("open", initialBalance))
	err := k.GetBankKeeper().MintCoins(ctx, types.ModuleName, initialDelegatorBalance)
	require.NoError(t, err)
	err = k.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, delAddr, initialDelegatorBalance)
	require.NoError(t, err)
}

func TestLock(t *testing.T) {
	k, ctx := setupKeeper(t)

	delAddr := sdk.AccAddress("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(2000)
	initializeDelegator(t, k, ctx, delAddr, initialDelegatorBalance)

	initialValidatorBalance := math.NewInt(1000)
	initializeValidator(t, k.GetStakingKeeper().(*keeper.Keeper), ctx, valAddr, initialValidatorBalance)

	amount := math.NewInt(1000)
	err = k.Lock(ctx, delAddr, valAddr, amount)
	require.NoError(t, err)

	// verify that lockup was added
	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, amount, lockedAmt)
}

func TestUnlock(t *testing.T) {
	k, ctx := setupKeeper(t)

	delAddr := sdk.AccAddress("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(2000)
	initializeDelegator(t, k, ctx, delAddr, initialDelegatorBalance)

	initialValidatorBalance := math.NewInt(1000)
	initializeValidator(t, k.GetStakingKeeper().(*keeper.Keeper), ctx, valAddr, initialValidatorBalance)

	amount := math.NewInt(1000)
	err = k.Lock(ctx, delAddr, valAddr, amount)
	require.NoError(t, err)

	// verify that lockup was added
	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, amount, lockedAmt)

	unbondTime, unlockTime, creationHeight, err := k.Unlock(ctx, delAddr, valAddr, math.NewInt(500))
	require.NoError(t, err)

	// verify that lockup was updated
	lockedAmt = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, math.ZeroInt(), lockedAmt)

	// check the unlocking entry
	found, amt, unbTime, unlTime := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.True(t, found)
	require.Equal(t, math.NewInt(500), amt)
	require.Equal(t, unbondTime, unbTime)
	require.Equal(t, unlockTime, unlTime)
}

func TestRedelegate(t *testing.T) {
	k, ctx := setupKeeper(t)

	delAddr := sdk.AccAddress("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	srcValAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	dstValAddr := sample.RandomValAddress()

	initialDelegatorBalance := math.NewInt(2000)
	initializeDelegator(t, k, ctx, delAddr, initialDelegatorBalance)

	initialValidatorBalance := math.NewInt(1000)
	initializeValidator(t, k.GetStakingKeeper().(*keeper.Keeper), ctx, srcValAddr, initialValidatorBalance)
	initializeValidator(t, k.GetStakingKeeper().(*keeper.Keeper), ctx, dstValAddr, initialValidatorBalance)

	// lock tokens with the source validator
	amount := math.NewInt(1000)
	require.NoError(t, k.Lock(ctx, delAddr, srcValAddr, amount))

	// redelegate from the source validator to the destination validator
	completionTime, err := k.Redelegate(ctx, delAddr, srcValAddr, dstValAddr, math.NewInt(500))
	require.NoError(t, err)

	// check lockup state
	srcLockup := k.GetLockupAmount(ctx, delAddr, srcValAddr)
	require.Equal(t, math.NewInt(500), srcLockup)

	dstLockup := k.GetLockupAmount(ctx, delAddr, dstValAddr)
	require.Equal(t, math.NewInt(500), dstLockup)

	// ensure completion time is set
	require.NotZero(t, completionTime)
}

func TestCompleteUnlocking(t *testing.T) {
	k, ctx := setupKeeper(t)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(200_000)
	initializeDelegator(t, k, ctx, delAddr, initialDelegatorBalance)

	initialValidatorBalance := math.NewInt(1_000_000)
	initializeValidator(t, k.GetStakingKeeper().(*keeper.Keeper), ctx, valAddr, initialValidatorBalance)

	lockAmount := math.NewInt(123_456)
	err = k.Lock(ctx, delAddr, valAddr, lockAmount)
	require.NoError(t, err)

	lockup := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, lockAmount, lockup)

	balance := k.GetBankKeeper().GetBalance(ctx, delAddr, appparams.DefaultBondDenom)
	require.Equal(t, initialDelegatorBalance.Sub(lockAmount), balance.Amount)

	// unlocking tokens
	unlockAmount := math.NewInt(123_456)
	adjustedUnlockAmount := unlockAmount.Sub(math.NewInt(1))
	unbondTime, unlockTime, creationHeight, err := k.Unlock(ctx, delAddr, valAddr, unlockAmount)
	require.NoError(t, err)

	lockup = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, math.NewInt(0), lockup)

	found, amt, unbTime, unlTime := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.True(t, found)
	require.Equal(t, adjustedUnlockAmount, amt)
	require.Equal(t, unbondTime, unbTime)
	require.Equal(t, unlockTime, unlTime)

	balance = k.GetBankKeeper().GetBalance(ctx, delAddr, appparams.DefaultBondDenom)
	require.Equal(t, initialDelegatorBalance.Sub(lockAmount), balance.Amount)

	// after 2 months
	ctx = ctx.WithBlockTime(sdk.UnwrapSDKContext(ctx).BlockTime().Add(60 * 24 * time.Hour))

	modAddr := authtypes.NewModuleAddress(types.ModuleName)
	k.GetStakingKeeper().CompleteUnbonding(ctx, modAddr, valAddr)

	found, amt, unbTime, unlTime = k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.True(t, found)
	require.Equal(t, adjustedUnlockAmount, amt)
	require.Equal(t, unbondTime, unbTime)
	require.Equal(t, unlockTime, unlTime)

	err = k.CompleteUnlocking(ctx)
	require.NoError(t, err)

	// verify that the balance is returned
	balance = k.GetBankKeeper().GetBalance(ctx, delAddr, appparams.DefaultBondDenom)
	require.Equal(t, initialDelegatorBalance.Sub(math.NewInt(1)), balance.Amount)
}

func TestCancelUnlocking(t *testing.T) {
	k, ctx := setupKeeper(t)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(200_000)
	initializeDelegator(t, k, ctx, delAddr, initialDelegatorBalance)

	initialValidatorBalance := math.NewInt(10_000_000)
	initializeValidator(t, k.GetStakingKeeper().(*keeper.Keeper), ctx, valAddr, initialValidatorBalance)

	initialAmount := math.NewInt(1000)
	unlockAmount := math.NewInt(500)
	adjustedUnlockAmount := unlockAmount.Sub(math.NewInt(1))

	// lock the initialAmount
	err = k.Lock(ctx, delAddr, valAddr, initialAmount)
	require.NoError(t, err)

	// verify that lockup was added
	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, initialAmount, lockedAmt)

	// unlock the unlockAmount
	unbondTime, unlockTime, creationHeight, err := k.Unlock(ctx, delAddr, valAddr, unlockAmount)
	require.NoError(t, err)

	// verify that lockup was updated
	lockedAmt = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, math.ZeroInt(), lockedAmt)

	// check the unlocking entry based on adjusted unlock amount
	found, amt, unbTime, unlTime := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.True(t, found)
	require.Equal(t, adjustedUnlockAmount, amt)
	require.Equal(t, unbondTime, unbTime)
	require.Equal(t, unlockTime, unlTime)

	err = k.CancelUnlocking(ctx, delAddr, valAddr, adjustedUnlockAmount)
	require.NoError(t, err)

	// verify that lockup was updated
	lockupAmount := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, math.NewInt(499), lockupAmount)

	// check the unlocking entry
	found, amt, unbTime, unlTime = k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.Equal(t, true, found)
	require.Equal(t, math.NewInt(499), amt)
	require.Equal(t, unbondTime, unbTime)
	require.Equal(t, unlockTime, unlTime)
}
