package keeper_test

import (
	"testing"
	"time"

	cosmosed25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	testutil "github.com/sourcenetwork/sourcehub/testutil"
	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
	"github.com/stretchr/testify/require"
)

// initializeValidator creates a validator and verifies that it was set correctly.
func initializeValidator(t *testing.T, k *stakingkeeper.Keeper, ctx sdk.Context, valAddr sdk.ValAddress, initialTokens math.Int) {
	validator := testutil.CreateTestValidator(t, ctx, k, valAddr, cosmosed25519.GenPrivKey().PubKey(), initialTokens)
	gotValidator, err := k.GetValidator(ctx, valAddr)
	require.Nil(t, err)
	require.Equal(t, validator.OperatorAddress, gotValidator.OperatorAddress)
}

// initializeDelegator initializes ba delegator with balance.
func initializeDelegator(t *testing.T, k *keeper.Keeper, ctx sdk.Context, delAddr sdk.AccAddress, initialBalance math.Int) {
	initialDelegatorBalance := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, initialBalance))
	err := k.GetBankKeeper().MintCoins(ctx, types.ModuleName, initialDelegatorBalance)
	require.NoError(t, err)
	err = k.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, delAddr, initialDelegatorBalance)
	require.NoError(t, err)
}

// TestLock verifies that a valid lockup is created on keeper.Lock().
func TestLock(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)

	amount := math.NewInt(1000)
	invalidAmount := math.NewInt(-100)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(2000)
	initializeDelegator(t, &k, ctx, delAddr, initialDelegatorBalance)
	initialValidatorBalance := math.NewInt(1000)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)

	// set initial block height and time
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	// locking invalid amounts should fail
	err = k.Lock(ctx, delAddr, valAddr, invalidAmount)
	require.Error(t, err)
	require.Contains(t, err.Error(), "lock non-positive amount")
	err = k.Lock(ctx, delAddr, valAddr, math.ZeroInt())
	require.Error(t, err)
	require.Contains(t, err.Error(), "lock non-positive amount")

	// lock valid amount
	err = k.Lock(ctx, delAddr, valAddr, amount)
	require.NoError(t, err)

	// verify that lockup was added
	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, amount, lockedAmt)
}

// TestUnlock verifies that a valid unlocking lockup is created on keeper.Unock().
func TestUnlock(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)

	lockAmount := math.NewInt(1000)
	unlockAmount := math.NewInt(500)
	invalidUnlockAmount := math.NewInt(-500)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(2000)
	initializeDelegator(t, &k, ctx, delAddr, initialDelegatorBalance)
	initialValidatorBalance := math.NewInt(1000)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)

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
	require.Contains(t, err.Error(), "unlock non-positive amount")
	_, _, _, err = k.Unlock(ctx, delAddr, valAddr, math.ZeroInt())
	require.Error(t, err)
	require.Contains(t, err.Error(), "unlock non-positive amount")

	creationHeight, completionTime, unlockTime, err := k.Unlock(ctx, delAddr, valAddr, unlockAmount)
	require.NoError(t, err)

	// verify that lockup was updated
	lockedAmt = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, lockAmount.Sub(unlockAmount), lockedAmt)

	// check the unlocking entry
	unlockingLockup := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.NotNil(t, unlockingLockup)
	require.Equal(t, unlockAmount, unlockingLockup.Amount)
	require.Equal(t, completionTime, unlockingLockup.CompletionTime)
	require.Equal(t, unlockTime, unlockingLockup.UnlockTime)
}

// TestRedelegate verifies that a locked amount is correctly redelegated on keeper.Redelegate().
func TestRedelegate(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)

	amount := math.NewInt(1000)
	invalidAmount := math.NewInt(-100)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	srcValAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)
	dstValAddr, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(2000)
	initializeDelegator(t, &k, ctx, delAddr, initialDelegatorBalance)
	initialValidatorBalance := math.NewInt(1000)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, srcValAddr, initialValidatorBalance)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, dstValAddr, initialValidatorBalance)

	// set initial block height and time
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	// lock tokens with the source validator
	require.NoError(t, k.Lock(ctx, delAddr, srcValAddr, amount))

	// redelegating invalid amounts should fail
	_, err = k.Redelegate(ctx, delAddr, srcValAddr, dstValAddr, invalidAmount)
	require.Error(t, err)
	require.Contains(t, err.Error(), "redelegate non-positive amount")
	_, err = k.Redelegate(ctx, delAddr, srcValAddr, dstValAddr, math.ZeroInt())
	require.Error(t, err)
	require.Contains(t, err.Error(), "redelegate non-positive amount")

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
	k, ctx := keepertest.TierKeeper(t)

	lockAmount := math.NewInt(123_456)
	unlockAmount := math.NewInt(123_456)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(200_000)
	initializeDelegator(t, &k, ctx, delAddr, initialDelegatorBalance)
	initialValidatorBalance := math.NewInt(1_000_000)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)

	// set initial block height and time
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	err = k.Lock(ctx, delAddr, valAddr, lockAmount)
	require.NoError(t, err)

	lockup := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, lockAmount, lockup)

	balance := k.GetBankKeeper().GetBalance(ctx, delAddr, appparams.DefaultBondDenom)
	require.Equal(t, initialDelegatorBalance.Sub(lockAmount), balance.Amount)

	// unlock tokens
	creationHeight, completionTime, unlockTime, err := k.Unlock(ctx, delAddr, valAddr, unlockAmount)
	require.NoError(t, err)

	lockup = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, math.ZeroInt(), lockup)

	unlockingLockup := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.NotNil(t, unlockingLockup)
	require.Equal(t, unlockAmount.Sub(math.OneInt()), unlockingLockup.Amount) // 123_455
	require.Equal(t, completionTime, unlockingLockup.CompletionTime)
	require.Equal(t, unlockTime, unlockingLockup.UnlockTime)

	balance = k.GetBankKeeper().GetBalance(ctx, delAddr, appparams.DefaultBondDenom)
	require.Equal(t, initialDelegatorBalance.Sub(lockAmount), balance.Amount)

	// completing unlocking lockup should be skipped if unlock time was not reached
	err = k.CompleteUnlocking(ctx)
	require.NoError(t, err)
	balance = k.GetBankKeeper().GetBalance(ctx, delAddr, appparams.DefaultBondDenom)
	require.Equal(t, initialDelegatorBalance.Sub(unlockAmount), balance.Amount)

	// advance block time by 60 days
	ctx = ctx.WithBlockHeight(3600 * 24 * 60).WithBlockTime(ctx.BlockTime().Add(60 * 24 * time.Hour))

	// completing unlocking lockup should be skipped if module balance is less than required amount
	err = k.CompleteUnlocking(ctx)
	require.NoError(t, err)
	balance = k.GetBankKeeper().GetBalance(ctx, delAddr, appparams.DefaultBondDenom)
	require.Equal(t, initialDelegatorBalance.Sub(unlockAmount), balance.Amount)

	// complete unbonding via the staking keeper
	modAddr := authtypes.NewModuleAddress(types.ModuleName)
	_, err = k.GetStakingKeeper().CompleteUnbonding(ctx, modAddr, valAddr)
	require.NoError(t, err)

	unlockingLockup = k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.NotNil(t, unlockingLockup)
	require.Equal(t, unlockAmount.Sub(math.OneInt()), unlockingLockup.Amount) // 123_455
	require.Equal(t, completionTime, unlockingLockup.CompletionTime)
	require.Equal(t, unlockTime, unlockingLockup.UnlockTime)

	// complete unlocking of matured unlocking lockups
	err = k.CompleteUnlocking(ctx)
	require.NoError(t, err)

	// verify that the balance is correct
	balance = k.GetBankKeeper().GetBalance(ctx, delAddr, appparams.DefaultBondDenom)
	require.Equal(t, initialDelegatorBalance.Sub(math.OneInt()), balance.Amount)
}

// TestCancelUnlocking verifies that the unlocking lockup is removed on keeper.CancelUnlocking().
func TestCancelUnlocking(t *testing.T) {
	k, ctx := keepertest.TierKeeper(t)

	initialLockAmount := math.NewInt(1000)
	updatedLockAmount := math.NewInt(700)
	unlockAmount := math.NewInt(500)
	partialUnlockAmount := math.NewInt(200)
	remainingUnlockAmount := math.NewInt(300)

	delAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(200_000)
	initializeDelegator(t, &k, ctx, delAddr, initialDelegatorBalance)
	initialValidatorBalance := math.NewInt(10_000_000)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, initialValidatorBalance)

	// set initial block height and time
	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	// lock the initialAmount
	err = k.Lock(ctx, delAddr, valAddr, initialLockAmount)
	require.NoError(t, err)

	// verify that lockup was added
	lockedAmt := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, initialLockAmount, lockedAmt)

	// unlock the unlockAmount (partial unlock)
	creationHeight, completionTime, unlockTime, err := k.Unlock(ctx, delAddr, valAddr, unlockAmount)
	require.NoError(t, err)

	// verify that lockup was updated
	lockedAmt = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, initialLockAmount.Sub(unlockAmount), lockedAmt) // 500

	// check the unlocking entry based on adjusted unlock amount
	unlockingLockup := k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.NotNil(t, unlockingLockup)
	require.Equal(t, unlockAmount.Sub(math.OneInt()), unlockingLockup.Amount) // 499
	require.Equal(t, completionTime, unlockingLockup.CompletionTime)
	require.Equal(t, unlockTime, unlockingLockup.UnlockTime)

	err = k.CancelUnlocking(ctx, delAddr, valAddr, creationHeight, math.NewInt(-100))
	require.Error(t, err)
	require.Contains(t, err.Error(), "cancel unlocking non-positive amount")
	err = k.CancelUnlocking(ctx, delAddr, valAddr, creationHeight, math.ZeroInt())
	require.Error(t, err)
	require.Contains(t, err.Error(), "cancel unlocking non-positive amount")

	// partially cancel the unlocking lockup
	err = k.CancelUnlocking(ctx, delAddr, valAddr, creationHeight, partialUnlockAmount)
	require.NoError(t, err)

	// verify that lockup was updated
	lockupAmount := k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, updatedLockAmount, lockupAmount) // 700

	// check the unlocking entry
	unlockingLockup = k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.NotNil(t, unlockingLockup)
	require.Equal(t, remainingUnlockAmount.Sub(math.OneInt()), unlockingLockup.Amount) // 299
	require.Equal(t, completionTime, unlockingLockup.CompletionTime)
	require.Equal(t, unlockTime, unlockingLockup.UnlockTime)

	// advance block height by 1 so that subsequent unlocking lockup is stored separately
	// otherwise, existing unlocking lockup is overrirden (e.g. delAddr/valAddr/creationHeight/)
	// TODO: handle edge case with 2+ messages at the same height
	ctx = ctx.WithBlockHeight(2).WithBlockTime(ctx.BlockTime().Add(time.Minute))

	// add new unlocking lockup record at height 2 to fully unlock the remaining adjustedUnlockAmountFinal
	creationHeight2, completionTime2, unlockTime2, err := k.Unlock(ctx, delAddr, valAddr, remainingUnlockAmount)
	require.NoError(t, err)

	// verify that lockup was updated
	lockedAmt = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, updatedLockAmount.Sub(remainingUnlockAmount), lockedAmt) // 400

	// check the unlocking entry based on adjusted unlock amount
	unlockingLockup = k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight2)
	require.NotNil(t, unlockingLockup)
	require.Equal(t, remainingUnlockAmount.Sub(math.OneInt()), unlockingLockup.Amount) // 299
	require.Equal(t, completionTime2, unlockingLockup.CompletionTime)
	require.Equal(t, unlockTime2, unlockingLockup.UnlockTime)

	// cancel (remove) the unlocking lockup at height 2
	err = k.CancelUnlocking(ctx, delAddr, valAddr, creationHeight2, remainingUnlockAmount.Sub(math.OneInt())) // 299
	require.NoError(t, err)

	// verify that lockup was updated
	lockupAmount = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, updatedLockAmount.Sub(math.OneInt()), lockupAmount) // 699

	// there is still a partial unlocking lockup at height 1 since we did not cancel it's whole amount
	unlockingLockup = k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.NotNil(t, unlockingLockup)
	require.Equal(t, remainingUnlockAmount.Sub(math.OneInt()), unlockingLockup.Amount) // 299
	require.Equal(t, completionTime, unlockingLockup.CompletionTime)
	require.Equal(t, unlockTime, unlockingLockup.UnlockTime)

	// cancel (remove) the remaining unlocking lockup at height 1
	err = k.CancelUnlocking(ctx, delAddr, valAddr, creationHeight, remainingUnlockAmount.Sub(math.OneInt())) // 299
	require.NoError(t, err)

	// verify that lockup was updated
	lockupAmount = k.GetLockupAmount(ctx, delAddr, valAddr)
	require.Equal(t, initialLockAmount.Sub(math.NewInt(2)), lockupAmount) // 998

	// confirm that unlocking lockup was removed if we cancel whole amount (e.g. use nil)
	unlockingLockup = k.GetUnlockingLockup(ctx, delAddr, valAddr, creationHeight)
	require.Nil(t, unlockingLockup)
}
