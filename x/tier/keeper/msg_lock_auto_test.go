package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	appparams "github.com/sourcenetwork/vera/app/params"
	keepertest "github.com/sourcenetwork/vera/testutil/keeper"
	epochstypes "github.com/sourcenetwork/vera/x/epochs/types"
	"github.com/sourcenetwork/vera/x/tier/types"
)

func TestMsgLockAuto(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	epoch := epochstypes.EpochInfo{
		Identifier:            types.EpochIdentifier,
		CurrentEpoch:          1,
		CurrentEpochStartTime: sdkCtx.BlockTime().Add(-5 * time.Minute),
		Duration:              5 * time.Minute,
	}
	k.GetEpochsKeeper().SetEpochInfo(ctx, epoch)

	validCoin1 := sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(100))
	validCoin2 := sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(3000))
	zeroCoin := sdk.NewCoin(appparams.DefaultBondDenom, math.ZeroInt())
	negativeAmount := math.NewInt(-1000)
	initialDelegatorBalance := math.NewInt(2000)
	initialValidatorBalance := math.NewInt(1000)

	delAddr := "vera1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s2dq2wz"
	valAddr1 := "veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0"
	valAddr2 := "veravaloper13fj7t2yptf9k6ad6fv38434znzay4s4pp0cewa"

	delAddress, err := sdk.AccAddressFromBech32(delAddr)
	require.NoError(t, err)
	valAddress1, err := sdk.ValAddressFromBech32(valAddr1)
	require.NoError(t, err)
	valAddress2, err := sdk.ValAddressFromBech32(valAddr2)
	require.NoError(t, err)

	keepertest.InitializeDelegator(t, &k, sdkCtx, delAddress, initialDelegatorBalance)
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddress1, initialValidatorBalance)
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddress2, initialValidatorBalance)

	testCases := []struct {
		name      string
		input     *types.MsgLockAuto
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid lock auto",
			input: &types.MsgLockAuto{
				DelegatorAddress: delAddr,
				Stake:            validCoin1,
			},
			expErr: false,
		},
		{
			name: "insufficient funds",
			input: &types.MsgLockAuto{
				DelegatorAddress: delAddr,
				Stake:            validCoin2,
			},
			expErr:    true,
			expErrMsg: "insufficient funds",
		},
		{
			name: "invalid stake amount (zero)",
			input: &types.MsgLockAuto{
				DelegatorAddress: delAddr,
				Stake:            zeroCoin,
			},
			expErr:    true,
			expErrMsg: "invalid amount",
		},
		{
			name: "invalid stake amount (negative)",
			input: &types.MsgLockAuto{
				DelegatorAddress: delAddr,
				Stake: sdk.Coin{
					Denom:  appparams.DefaultBondDenom,
					Amount: negativeAmount,
				},
			},
			expErr:    true,
			expErrMsg: "invalid amount",
		},
		{
			name: "invalid delegator address",
			input: &types.MsgLockAuto{
				DelegatorAddress: "invalid-delegator-address",
				Stake:            validCoin1,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.ValidateBasic()
			if err != nil {
				if tc.expErr {
					require.Contains(t, err.Error(), tc.expErrMsg)
					return
				}
				t.Fatalf("unexpected error in ValidateBasic: %v", err)
			}

			resp, err := ms.LockAuto(sdkCtx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.NotEmpty(t, resp.ValidatorAddress)

				// Verify the response contains a valid validator address
				_, err := sdk.ValAddressFromBech32(resp.ValidatorAddress)
				require.NoError(t, err)

				// Verify it's one of the bonded validators
				require.True(t, resp.ValidatorAddress == valAddr1 || resp.ValidatorAddress == valAddr2)
			}
		})
	}
}

// TestMsgLockAutoValidatorSelection tests the validator selection logic in LockAuto.
func TestMsgLockAutoValidatorSelection(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	epoch := epochstypes.EpochInfo{
		Identifier:            types.EpochIdentifier,
		CurrentEpoch:          1,
		CurrentEpochStartTime: sdkCtx.BlockTime().Add(-5 * time.Minute),
		Duration:              5 * time.Minute,
	}
	k.GetEpochsKeeper().SetEpochInfo(ctx, epoch)

	delAddr := "vera1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s2dq2wz"
	valAddr1 := "veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0"
	valAddr2 := "veravaloper13fj7t2yptf9k6ad6fv38434znzay4s4pp0cewa"

	delAddress, err := sdk.AccAddressFromBech32(delAddr)
	require.NoError(t, err)
	valAddress1, err := sdk.ValAddressFromBech32(valAddr1)
	require.NoError(t, err)
	valAddress2, err := sdk.ValAddressFromBech32(valAddr2)
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(5000)
	initialValidatorBalance := math.NewInt(1000)

	keepertest.InitializeDelegator(t, &k, sdkCtx, delAddress, initialDelegatorBalance)
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddress1, initialValidatorBalance)
	keepertest.InitializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddress2, initialValidatorBalance)

	t.Run("first lock selects validator with smallest tier module delegation", func(t *testing.T) {
		// First lock should select validator with smallest delegation from tier module
		// Both validators have 0 delegation from tier module initially
		resp, err := ms.LockAuto(sdkCtx, &types.MsgLockAuto{
			DelegatorAddress: delAddr,
			Stake:            sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(100)),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		firstSelectedVal := resp.ValidatorAddress
		require.True(t, firstSelectedVal == valAddr1 || firstSelectedVal == valAddr2)

		// Verify lockup was created
		firstValAddr, err := sdk.ValAddressFromBech32(firstSelectedVal)
		require.NoError(t, err)
		lockup := k.GetLockup(ctx, delAddress, firstValAddr)
		require.NotNil(t, lockup)
		require.Equal(t, math.NewInt(100), lockup.Amount)
	})

	t.Run("second lock to same developer selects validator with smallest existing lockup", func(t *testing.T) {
		// Lock 200 more, should go to the same validator as first lock (smallest existing lockup)
		resp, err := ms.LockAuto(sdkCtx, &types.MsgLockAuto{
			DelegatorAddress: delAddr,
			Stake:            sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(200)),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Get the first selected validator by checking both possible validators
		var firstSelectedVal sdk.ValAddress
		lockup1 := k.GetLockup(ctx, delAddress, valAddress1)
		lockup2 := k.GetLockup(ctx, delAddress, valAddress2)

		if lockup1 != nil && lockup2 == nil {
			firstSelectedVal = valAddress1
		} else if lockup2 != nil && lockup1 == nil {
			firstSelectedVal = valAddress2
		} else if lockup1 != nil && lockup2 != nil {
			// Both have lockups, find the one with smaller amount (from first lock)
			if lockup1.Amount.LT(lockup2.Amount) {
				firstSelectedVal = valAddress1
			} else {
				firstSelectedVal = valAddress2
			}
		}

		// Second lock should go to same validator (it had the smallest lockup)
		secondValAddr, err := sdk.ValAddressFromBech32(resp.ValidatorAddress)
		require.NoError(t, err)
		require.True(t, secondValAddr.Equals(firstSelectedVal))

		// Verify lockup was updated
		lockup := k.GetLockup(ctx, delAddress, secondValAddr)
		require.NotNil(t, lockup)
		require.Equal(t, math.NewInt(300), lockup.Amount) // 100 + 200
	})

	t.Run("lock to different validator then select smallest for third lock", func(t *testing.T) {
		// Find which validator has the existing lockup
		lockup1 := k.GetLockup(ctx, delAddress, valAddress1)
		lockup2 := k.GetLockup(ctx, delAddress, valAddress2)

		var existingVal, otherVal sdk.ValAddress
		if lockup1 != nil && lockup2 == nil {
			existingVal = valAddress1
			otherVal = valAddress2
		} else if lockup2 != nil && lockup1 == nil {
			existingVal = valAddress2
			otherVal = valAddress1
		} else {
			// If both have lockups, find the smaller one
			if lockup1.Amount.LT(lockup2.Amount) {
				existingVal = valAddress1
				otherVal = valAddress2
			} else {
				existingVal = valAddress2
				otherVal = valAddress1
			}
		}

		// Lock a large amount to the other validator manually
		err = k.Lock(ctx, delAddress, otherVal, math.NewInt(500))
		require.NoError(t, err)

		// Now auto lock should select the validator with smallest lockup (300 < 500)
		resp, err := ms.LockAuto(sdkCtx, &types.MsgLockAuto{
			DelegatorAddress: delAddr,
			Stake:            sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(50)),
		})
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Should select the validator with 300 (existing one)
		selectedVal, err := sdk.ValAddressFromBech32(resp.ValidatorAddress)
		require.NoError(t, err)
		require.True(t, selectedVal.Equals(existingVal))

		// Verify the lockup amount
		lockup := k.GetLockup(ctx, delAddress, existingVal)
		require.NotNil(t, lockup)
		require.Equal(t, math.NewInt(350), lockup.Amount) // 300 + 50
	})
}
