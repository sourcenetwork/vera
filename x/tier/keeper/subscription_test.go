package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
	"github.com/stretchr/testify/require"
)

func TestGetAndSetDeveloper(t *testing.T) {
	k, ctx := setupKeeper(t)

	developerAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)

	developer := k.GetDeveloper(ctx, developerAddr)
	require.Nil(t, developer)

	expectedDeveloper := &types.Developer{
		Address:         developerAddr.String(),
		AutoLockEnabled: true,
	}
	k.setDeveloper(ctx, developerAddr, expectedDeveloper)

	retrievedDeveloper := k.GetDeveloper(ctx, developerAddr)
	require.NotNil(t, retrievedDeveloper)
	require.Equal(t, expectedDeveloper.Address, retrievedDeveloper.Address)
	require.Equal(t, expectedDeveloper.AutoLockEnabled, retrievedDeveloper.AutoLockEnabled)
}

func TestRemoveDeveloper(t *testing.T) {
	k, ctx := setupKeeper(t)

	developerAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)

	developer := &types.Developer{
		Address:         developerAddr.String(),
		AutoLockEnabled: false,
	}
	k.setDeveloper(ctx, developerAddr, developer)

	retrievedDeveloper := k.GetDeveloper(ctx, developerAddr)
	require.NotNil(t, retrievedDeveloper)

	k.removeDeveloper(ctx, developerAddr)

	retrievedDeveloper = k.GetDeveloper(ctx, developerAddr)
	require.Nil(t, retrievedDeveloper)
}

func TestGetAndSetUserSubscription(t *testing.T) {
	k, ctx := setupKeeper(t)

	developerAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	userAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)

	userSub := k.GetUserSubscription(ctx, developerAddr, userAddr)
	require.Nil(t, userSub)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()
	creditAmount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
	expectedUserSub := &types.UserSubscription{
		Developer:    developerAddr.String(),
		User:         userAddr.String(),
		CreditAmount: creditAmount,
		Period:       3600,
		StartDate:    now,
		LastRenewed:  now,
	}

	k.setUserSubscription(ctx, developerAddr, userAddr, expectedUserSub)

	retrievedUserSub := k.GetUserSubscription(ctx, developerAddr, userAddr)
	require.NotNil(t, retrievedUserSub)
	require.Equal(t, expectedUserSub.Developer, retrievedUserSub.Developer)
	require.Equal(t, expectedUserSub.User, retrievedUserSub.User)
	require.Equal(t, expectedUserSub.CreditAmount, retrievedUserSub.CreditAmount)
	require.Equal(t, expectedUserSub.Period, retrievedUserSub.Period)
	require.True(t, expectedUserSub.StartDate.Equal(retrievedUserSub.StartDate))
	require.True(t, expectedUserSub.LastRenewed.Equal(retrievedUserSub.LastRenewed))
}

func TestRemoveUserSubscription(t *testing.T) {
	k, ctx := setupKeeper(t)

	developerAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	userAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)

	creditAmount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(500))
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()
	userSub := &types.UserSubscription{
		Developer:    developerAddr.String(),
		User:         userAddr.String(),
		CreditAmount: creditAmount,
		Period:       7200,
		StartDate:    now,
		LastRenewed:  now,
	}
	k.setUserSubscription(ctx, developerAddr, userAddr, userSub)

	retrievedUserSub := k.GetUserSubscription(ctx, developerAddr, userAddr)
	require.NotNil(t, retrievedUserSub)

	k.removeUserSubscription(ctx, developerAddr, userAddr)

	retrievedUserSub = k.GetUserSubscription(ctx, developerAddr, userAddr)
	require.Nil(t, retrievedUserSub)
}

func TestMustIterateUserSubscriptions(t *testing.T) {
	k, ctx := setupKeeper(t)

	dev1Addr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	dev2Addr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	user1Addr, err := sdk.AccAddressFromBech32("source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4")
	require.NoError(t, err)
	user2Addr, err := sdk.AccAddressFromBech32("source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy")
	require.NoError(t, err)

	creditAmount1 := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
	creditAmount2 := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(2000))
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	userSub1 := &types.UserSubscription{
		Developer:    dev1Addr.String(),
		User:         user1Addr.String(),
		CreditAmount: creditAmount1,
		Period:       3600,
		StartDate:    now,
		LastRenewed:  now,
	}

	userSub2 := &types.UserSubscription{
		Developer:    dev2Addr.String(),
		User:         user2Addr.String(),
		CreditAmount: creditAmount2,
		Period:       7200,
		StartDate:    now,
		LastRenewed:  now,
	}

	k.setUserSubscription(ctx, dev1Addr, user1Addr, userSub1)
	k.setUserSubscription(ctx, dev2Addr, user2Addr, userSub2)

	var foundSubscriptions []types.UserSubscription
	k.mustIterateUserSubscriptions(ctx, func(developerAddr sdk.AccAddress, userAddr sdk.AccAddress, userSubscription types.UserSubscription) {
		foundSubscriptions = append(foundSubscriptions, userSubscription)
	})

	require.Len(t, foundSubscriptions, 2)

	found1, found2 := false, false
	for _, sub := range foundSubscriptions {
		if sub.Developer == dev1Addr.String() && sub.User == user1Addr.String() {
			found1 = true
			require.Equal(t, creditAmount1, sub.CreditAmount)
		}
		if sub.Developer == dev2Addr.String() && sub.User == user2Addr.String() {
			found2 = true
			require.Equal(t, creditAmount2, sub.CreditAmount)
		}
	}
	require.True(t, found1, "First subscription not found")
	require.True(t, found2, "Second subscription not found")
}

func TestMustIterateUserSubscriptionsForDeveloper(t *testing.T) {
	k, ctx := setupKeeper(t)

	dev1Addr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	dev2Addr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	user1Addr, err := sdk.AccAddressFromBech32("source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4")
	require.NoError(t, err)
	user2Addr, err := sdk.AccAddressFromBech32("source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy")
	require.NoError(t, err)

	creditAmount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()

	userSub1 := &types.UserSubscription{
		Developer:    dev1Addr.String(),
		User:         user1Addr.String(),
		CreditAmount: creditAmount,
		Period:       3600,
		StartDate:    now,
		LastRenewed:  now,
	}

	userSub2 := &types.UserSubscription{
		Developer:    dev1Addr.String(),
		User:         user2Addr.String(),
		CreditAmount: creditAmount,
		Period:       3600,
		StartDate:    now,
		LastRenewed:  now,
	}

	userSub3 := &types.UserSubscription{
		Developer:    dev2Addr.String(),
		User:         user1Addr.String(),
		CreditAmount: creditAmount,
		Period:       3600,
		StartDate:    now,
		LastRenewed:  now,
	}

	k.setUserSubscription(ctx, dev1Addr, user1Addr, userSub1)
	k.setUserSubscription(ctx, dev1Addr, user2Addr, userSub2)
	k.setUserSubscription(ctx, dev2Addr, user1Addr, userSub3)

	var foundSubscriptions []types.UserSubscription
	k.mustIterateUserSubscriptionsForDeveloper(ctx, dev1Addr, func(developerAddr sdk.AccAddress, userAddr sdk.AccAddress, userSubscription types.UserSubscription) {
		foundSubscriptions = append(foundSubscriptions, userSubscription)
	})

	require.Len(t, foundSubscriptions, 2, "Should find exactly 2 subscriptions for dev1")

	for _, sub := range foundSubscriptions {
		require.Equal(t, dev1Addr.String(), sub.Developer)
	}
}

func TestGetTotalDevGranted(t *testing.T) {
	k, ctx := setupKeeper(t)

	developerAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)

	total, err := k.getTotalDevGranted(ctx, developerAddr)
	require.NoError(t, err)
	require.True(t, total.IsZero())

	amount := math.NewInt(1500)
	err = k.updateDeveloperTotalGranted(ctx, developerAddr, amount, true)
	require.NoError(t, err)

	total, err = k.getTotalDevGranted(ctx, developerAddr)
	require.NoError(t, err)
	require.Equal(t, amount, total)
}

func TestUpdateDeveloperTotalGranted(t *testing.T) {
	k, ctx := setupKeeper(t)

	developerAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)

	total, err := k.getTotalDevGranted(ctx, developerAddr)
	require.NoError(t, err)
	require.True(t, total.IsZero())

	err = k.updateDeveloperTotalGranted(ctx, developerAddr, math.NewInt(1000), true)
	require.NoError(t, err)

	total, err = k.getTotalDevGranted(ctx, developerAddr)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(1000), total)

	err = k.updateDeveloperTotalGranted(ctx, developerAddr, math.NewInt(500), true)
	require.NoError(t, err)

	total, err = k.getTotalDevGranted(ctx, developerAddr)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(1500), total)

	err = k.updateDeveloperTotalGranted(ctx, developerAddr, math.NewInt(300), false)
	require.NoError(t, err)

	total, err = k.getTotalDevGranted(ctx, developerAddr)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(1200), total)

	err = k.updateDeveloperTotalGranted(ctx, developerAddr, math.NewInt(2000), false)
	require.Error(t, err)
	require.ErrorContains(t, err, "subtract")
}

func TestValidateDeveloperCredits(t *testing.T) {
	k, ctx := setupKeeper(t)

	developerAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initializeDelegator(t, &k, ctx, developerAddr, math.NewInt(1400))
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, math.NewInt(1_000_000))

	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	require.NoError(t, k.Lock(ctx, developerAddr, valAddr, math.NewInt(1400)))

	err = k.validateDeveloperCredits(ctx, developerAddr, math.NewInt(1000))
	require.NoError(t, err, "Should pass when developer has enough credits")

	err = k.updateDeveloperTotalGranted(ctx, developerAddr, math.NewInt(1500), true)
	require.NoError(t, err)

	err = k.validateDeveloperCredits(ctx, developerAddr, math.NewInt(300))
	require.NoError(t, err, "Should pass when requesting within available credits")

	err = k.validateDeveloperCredits(ctx, developerAddr, math.NewInt(600))
	require.Error(t, err)
	require.ErrorContains(t, err, "insufficient available credits")
}

func TestGrantPeriodicAllowance(t *testing.T) {
	k, ctx := setupKeeper(t)

	granterAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	granteeAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	createAccount(t, &k, ctx, granterAddr)
	createAccount(t, &k, ctx, granteeAddr)

	initializeDelegator(t, &k, ctx, granterAddr, math.NewInt(1000))
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, math.NewInt(1_000_000))

	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	spendLimit := sdk.NewCoins(sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(100)))
	period := time.Hour

	err = k.grantPeriodicAllowance(ctx, granterAddr, granteeAddr, spendLimit, period)
	require.NoError(t, err)

	allowance, err := k.feegrantKeeper.GetAllowance(ctx, granterAddr, granteeAddr)
	require.NoError(t, err)
	require.NotNil(t, allowance)

	periodicAllowance, ok := allowance.(*feegrant.PeriodicAllowance)
	require.True(t, ok, "Expected PeriodicAllowance")
	require.Equal(t, spendLimit, periodicAllowance.Basic.SpendLimit)
	require.Equal(t, period, periodicAllowance.Period)
}

func TestExpireAllowance(t *testing.T) {
	k, ctx := setupKeeper(t)

	granterAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	granteeAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	createAccount(t, &k, ctx, granterAddr)
	createAccount(t, &k, ctx, granteeAddr)

	initializeDelegator(t, &k, ctx, granterAddr, math.NewInt(1000))
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, math.NewInt(1_000_000))

	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	spendLimit := sdk.NewCoins(sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(100)))
	period := time.Hour
	err = k.grantPeriodicAllowance(ctx, granterAddr, granteeAddr, spendLimit, period)
	require.NoError(t, err)

	allowance, err := k.feegrantKeeper.GetAllowance(ctx, granterAddr, granteeAddr)
	require.NoError(t, err)
	require.NotNil(t, allowance)

	err = k.expireAllowance(ctx, granterAddr, granteeAddr)
	require.NoError(t, err)

	expiredAllowance, err := k.feegrantKeeper.GetAllowance(ctx, granterAddr, granteeAddr)
	require.NoError(t, err)
	require.NotNil(t, expiredAllowance)

	basicAllowance, ok := expiredAllowance.(*feegrant.BasicAllowance)
	require.True(t, ok, "Expected BasicAllowance after expiration")
	require.NotNil(t, basicAllowance.Expiration)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	require.True(t, basicAllowance.Expiration.Equal(sdkCtx.BlockTime()) || basicAllowance.Expiration.Before(sdkCtx.BlockTime()))
}

func TestCheckAndAutoLockDeveloperCredits(t *testing.T) {
	k, ctx := setupKeeper(t)

	developerAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	userAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initializeDelegator(t, &k, ctx, developerAddr, math.NewInt(1000))
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, math.NewInt(1_000_000))

	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	// Set up a user subscription that requires more credits than available
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()
	creditAmount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(2000))
	userSub := &types.UserSubscription{
		Developer:    developerAddr.String(),
		User:         userAddr.String(),
		CreditAmount: creditAmount,
		Period:       3600,
		StartDate:    now,
		LastRenewed:  now,
	}
	k.setUserSubscription(ctx, developerAddr, userAddr, userSub)

	err = k.updateDeveloperTotalGranted(ctx, developerAddr, creditAmount.Amount, true)
	require.NoError(t, err)

	developer := &types.Developer{
		Address:         developerAddr.String(),
		AutoLockEnabled: false,
	}
	k.setDeveloper(ctx, developerAddr, developer)

	err = k.checkAndAutoLockDeveloperCredits(ctx, 1)
	require.NoError(t, err)

	// Check that an event was emitted for insufficient credits
	sdkCtx = sdk.UnwrapSDKContext(ctx)
	events := sdkCtx.EventManager().Events()
	foundEvent := false
	for _, event := range events {
		if event.Type == "developer_insufficient_credits" {
			foundEvent = true
			break
		}
	}
	require.True(t, foundEvent, "Should emit developer_insufficient_credits event")
}

func TestAutoLockDeveloperCredits(t *testing.T) {
	k, ctx := setupKeeper(t)

	developerAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initializeDelegator(t, &k, ctx, developerAddr, math.NewInt(1000))
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, math.NewInt(1_000_000))

	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	developer := &types.Developer{
		Address:         developerAddr.String(),
		AutoLockEnabled: true,
	}

	err = k.autoLockDeveloperCredits(ctx, developerAddr, developer, math.ZeroInt())
	require.Error(t, err)
	require.ErrorContains(t, err, "auto-lock amount must be positive")

	err = k.autoLockDeveloperCredits(ctx, developerAddr, developer, math.NewInt(-100))
	require.Error(t, err)
	require.ErrorContains(t, err, "auto-lock amount must be positive")

}

func TestAddLockupForRegistration(t *testing.T) {
	k, ctx := setupKeeper(t)

	developerAddr, err := sdk.AccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)

	initializeDelegator(t, &k, ctx, developerAddr, math.NewInt(500))

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddr, math.NewInt(1_000_000))

	ctx = ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	requiredAmount := math.NewInt(300)
	err = k.addLockupForRegistration(ctx, developerAddr, requiredAmount)
	require.NoError(t, err)

	requiredAmount = math.NewInt(1000)

	// We need 670 uopen to get 1000 ucredit with current rate, but developer only has 200 uopen
	err = k.addLockupForRegistration(ctx, developerAddr, requiredAmount)
	require.Error(t, err)
	require.ErrorContains(t, err, "insufficient funds")

	initializeDelegator(t, &k, ctx, developerAddr, math.NewInt(470)) // 200 + 470 = 670
	err = k.addLockupForRegistration(ctx, developerAddr, requiredAmount)
	require.NoError(t, err)
}
