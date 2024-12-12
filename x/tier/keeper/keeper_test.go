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

// TestLock verifies that a valid lockup is created on keeper.Lock().
func TestLock(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	amount := math.NewInt(1000)
	invalidAmount := math.NewInt(-100)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(2000)
	initializeDelegator(t, k, ctx, delAddr, initialDelegatorBalance)
	initialValidatorBalance := math.NewInt(1000)
	initializeValidator(t, k.GetStakingKeeper().(*keeper.Keeper), ctx, valAddr, initialValidatorBalance)

	// set initial block height and time
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	// locking invalid amounts should fail
	err = k.Lock(ctx, delAddr, valAddr, invalidAmount)
	require.Error(t, err)
	err = k.Lock(ctx, delAddr, valAddr, math.ZeroInt())
	require.Error(t, err)

	// lock valid amount
	err = k.Lock(ctx, delAddr, valAddr, amount)
	require.NoError(t, err)

	// verify that lockup was added
	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, amount, lockedAmt)
}

// TestUnlock verifies that a valid unlocking lockup is created on keeper.Unock().
func TestUnlock(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	lockAmount := math.NewInt(1000)
	unlockAmount := math.NewInt(500)
	invalidUnlockAmount := math.NewInt(-500)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(2000)
	initializeDelegator(t, k, ctx, delAddr, initialDelegatorBalance)
	initialValidatorBalance := math.NewInt(1000)
	initializeValidator(t, k.GetStakingKeeper().(*keeper.Keeper), ctx, valAddr, initialValidatorBalance)

	// set initial block height and time
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	err = k.Lock(ctx, delAddr, valAddr, lockAmount)
	require.NoError(t, err)

	// verify that lockup was added
	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, lockAmount, lockedAmt)

	// unlocking invalid amounts should fail
	_, _, _, err = k.Unlock(ctx, delAddr, valAddr, invalidUnlockAmount)
	require.Error(t, err)
	_, _, _, err = k.Unlock(ctx, delAddr, valAddr, math.ZeroInt())
	require.Error(t, err)

	unbondTime, unlockTime, creationHeight, err := k.Unlock(ctx, delAddr, valAddr, unlockAmount)
	require.NoError(t, err)

	// verify that lockup was updated
	lockedAmt = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, lockAmount.Sub(unlockAmount), lockedAmt)

	// check the unlocking entry
	found, amt, unbTime, unlTime := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.True(t, found)
	require.Equal(t, unlockAmount, amt)
	require.Equal(t, unbondTime, unbTime)
	require.Equal(t, unlockTime, unlTime)
}

// TestRedelegate verifies that a locked amount is correctly redelegated on keeper.Redelegate().
func TestRedelegate(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	amount := math.NewInt(1000)
	invalidAmount := math.NewInt(-100)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	srcValAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)
	dstValAddr, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(2000)
	initializeDelegator(t, k, ctx, delAddr, initialDelegatorBalance)
	initialValidatorBalance := math.NewInt(1000)
	initializeValidator(t, k.GetStakingKeeper().(*keeper.Keeper), ctx, srcValAddr, initialValidatorBalance)
	initializeValidator(t, k.GetStakingKeeper().(*keeper.Keeper), ctx, dstValAddr, initialValidatorBalance)

	// set initial block height and time
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	// lock tokens with the source validator
	require.NoError(t, k.Lock(ctx, delAddr, srcValAddr, amount))

	// redelegating invalid amounts should fail
	_, err = k.Redelegate(ctx, delAddr, srcValAddr, dstValAddr, invalidAmount)
	require.Error(t, err)
	_, err = k.Redelegate(ctx, delAddr, srcValAddr, dstValAddr, math.ZeroInt())
	require.Error(t, err)

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

// TestCompleteUnlocking verifies that 'fully unlocked' unlocking lockups are removed on keeper.CompleteUnlocking().
// Block time is advanced by 60 days from when keeper.Unlock() is called to make sure that the unlock time is in the past.
func TestCompleteUnlocking(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	lockAmount := math.NewInt(123_456)
	unlockAmount := math.NewInt(123_456)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(200_000)
	initializeDelegator(t, k, ctx, delAddr, initialDelegatorBalance)
	initialValidatorBalance := math.NewInt(1_000_000)
	initializeValidator(t, k.GetStakingKeeper().(*keeper.Keeper), ctx, valAddr, initialValidatorBalance)

	// set initial block height and time
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	err = k.Lock(ctx, delAddr, valAddr, lockAmount)
	require.NoError(t, err)

	lockup := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, lockAmount, lockup)

	balance := k.GetBankKeeper().GetBalance(ctx, delAddr, appparams.DefaultBondDenom)
	require.Equal(t, initialDelegatorBalance.Sub(lockAmount), balance.Amount)

	adjustedUnlockAmount := unlockAmount.Sub(math.OneInt())

	// unlock tokens
	unbondTime, unlockTime, creationHeight, err := k.Unlock(ctx, delAddr, valAddr, unlockAmount)
	require.NoError(t, err)

	lockup = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, math.ZeroInt(), lockup)

	found, amt, unbTime, unlTime := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.True(t, found)
	require.Equal(t, adjustedUnlockAmount, amt)
	require.Equal(t, unbondTime, unbTime)
	require.Equal(t, unlockTime, unlTime)

	balance = k.GetBankKeeper().GetBalance(ctx, delAddr, appparams.DefaultBondDenom)
	require.Equal(t, initialDelegatorBalance.Sub(lockAmount), balance.Amount)

	// advance block time by 60 days
	ctx = ctx.WithBlockTime(sdk.UnwrapSDKContext(ctx).BlockTime().Add(60 * 24 * time.Hour))

	// complete unbonding via the staking keeper
	modAddr := authtypes.NewModuleAddress(types.ModuleName)
	_, err = k.GetStakingKeeper().CompleteUnbonding(ctx, modAddr, valAddr)
	require.NoError(t, err)

	found, amt, unbTime, unlTime = k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.True(t, found)
	require.Equal(t, adjustedUnlockAmount, amt)
	require.Equal(t, unbondTime, unbTime)
	require.Equal(t, unlockTime, unlTime)

	// complete unlocking of matured unlocking lockups
	err = k.CompleteUnlocking(ctx)
	require.NoError(t, err)

	// verify that the balance is correct
	balance = k.GetBankKeeper().GetBalance(ctx, delAddr, appparams.DefaultBondDenom)
	require.Equal(t, initialDelegatorBalance.Sub(math.OneInt()), balance.Amount)
}

// TestCancelUnlocking verifies that the unlocking lockup is removed on keeper.CancelUnlocking().
func TestCancelUnlocking(t *testing.T) {
	k, ctx := testutil.SetupKeeper(t)

	initialAmount := math.NewInt(1000)
	unlockAmount := math.NewInt(500)
	partialUnlockAmount := math.NewInt(200)
	adjustedInitialAmount := initialAmount.Sub(math.OneInt()) // 999
	adjustedUnlockAmount := unlockAmount.Sub(math.OneInt())   // 499
	newLockAmount := initialAmount.Sub(unlockAmount).Add(partialUnlockAmount)
	adjustedUnlockAmountFinal := initialAmount.Sub(unlockAmount).Sub(partialUnlockAmount).Sub(math.OneInt())

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(200_000)
	initializeDelegator(t, k, ctx, delAddr, initialDelegatorBalance)
	initialValidatorBalance := math.NewInt(10_000_000)
	initializeValidator(t, k.GetStakingKeeper().(*keeper.Keeper), ctx, valAddr, initialValidatorBalance)

	// set initial block height and time
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	// lock the initialAmount
	err = k.Lock(ctx, delAddr, valAddr, initialAmount)
	require.NoError(t, err)

	// verify that lockup was added
	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, initialAmount, lockedAmt)

	// unlock the unlockAmount (partial unlock)
	unbondTime, unlockTime, creationHeight, err := k.Unlock(ctx, delAddr, valAddr, unlockAmount)
	require.NoError(t, err)

	// verify that lockup was updated
	lockedAmt = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, initialAmount.Sub(unlockAmount), lockedAmt) // 500

	// check the unlocking entry based on adjusted unlock amount
	found, amt, unbTime, unlTime := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.True(t, found)
	require.Equal(t, adjustedUnlockAmount, amt) // 499
	require.Equal(t, unbondTime, unbTime)
	require.Equal(t, unlockTime, unlTime)

	// partially cancel the unlocking lockup
	err = k.CancelUnlocking(ctx, delAddr, valAddr, creationHeight, &partialUnlockAmount)
	require.NoError(t, err)

	// verify that lockup was updated
	lockupAmount := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, newLockAmount, lockupAmount) // 700

	// check the unlocking entry
	found, amt, unbTime, unlTime = k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.Equal(t, true, found)
	require.Equal(t, adjustedUnlockAmountFinal, amt) // 299
	require.Equal(t, unbondTime, unbTime)
	require.Equal(t, unlockTime, unlTime)

	// advance block height by 1 so that subsequent unlocking lockup is stored separately
	// otherwise, existing unlocking lockup is overrirden (e.g. delAddr/valAddr/creationHeight/)
	// TODO: handle edge case with 2+ messages at the same height
	ctx = ctx.WithBlockHeight(2).WithBlockTime(ctx.BlockTime().Add(time.Minute))

	// add new unlocking lockup record at height 2 to fully unlock the remaining adjustedUnlockAmountFinal
	unbondTime2, unlockTime2, creationHeight2, err := k.Unlock(ctx, delAddr, valAddr, adjustedUnlockAmountFinal)
	require.NoError(t, err)

	// verify that lockup was updated
	lockedAmt = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, newLockAmount.Sub(adjustedUnlockAmountFinal), lockedAmt) // 401

	// check the unlocking entry based on adjusted unlock amount
	found, amt, unbTime, unlTime = k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight2)
	require.True(t, found)
	require.Equal(t, adjustedUnlockAmountFinal.Sub(math.OneInt()), amt) // 298
	require.Equal(t, unbondTime2, unbTime)
	require.Equal(t, unlockTime2, unlTime)

	// cancel (remove) the unlocking lockup at height 2
	err = k.CancelUnlocking(ctx, delAddr, valAddr, creationHeight2, nil)
	require.NoError(t, err)

	// verify that lockup was updated
	lockupAmount = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, newLockAmount.Sub(math.OneInt()), lockupAmount) // 699

	// there is still a partial unlocking lockup at height 1 since we did not cancel it's whole amount
	found, amt, unbTime, unlTime = k.GetUnlockingLockup(ctx, delAddr, valAddr, 1)
	require.Equal(t, true, found)
	require.Equal(t, adjustedUnlockAmountFinal, amt) // 299
	require.Equal(t, unbondTime, unbTime)
	require.Equal(t, unlockTime, unlTime)

	// cancel (remove) the remaining unlocking lockup at height 1
	err = k.CancelUnlocking(ctx, delAddr, valAddr, 1, nil)
	require.NoError(t, err)

	// verify that lockup was updated
	lockupAmount = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, adjustedInitialAmount.Sub(math.OneInt()), lockupAmount) // 998

	// confirm that unlocking lockup was removed if we cancel whole amount (e.g. use nil)
	found, amt, unbTime, unlTime = k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.Equal(t, false, found)
	require.Equal(t, math.ZeroInt(), amt)
	require.Equal(t, time.Time{}, unbTime)
	require.Equal(t, time.Time{}, unlTime)
}
