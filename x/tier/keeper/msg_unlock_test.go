package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func TestMsgUnlock(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	validCoin := sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(100))
	zeroCoin := sdk.NewCoin(appparams.DefaultBondDenom, math.ZeroInt())
	negativeAmount := math.NewInt(-100)
	initialDelegatorBalance := math.NewInt(2000)
	initialValidatorBalance := math.NewInt(1000)

	delAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	valAddr := "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm"

	delAddress, err := sdk.AccAddressFromBech32(delAddr)
	require.NoError(t, err)
	valAddress, err := sdk.ValAddressFromBech32(valAddr)
	require.NoError(t, err)

	initializeDelegator(t, &k, sdkCtx, delAddress, initialDelegatorBalance)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddress, initialValidatorBalance)

	// lock some tokens to test the unlock logic
	err = k.Lock(ctx, delAddress, valAddress, validCoin.Amount)
	require.NoError(t, err)

	// expectedUnlockTime should match the SetLockup logic for setting the unlock time
	params := k.GetParams(ctx)
	epochDuration := *params.EpochDuration
	expectedUnlockTime := sdkCtx.BlockTime().Add(epochDuration * time.Duration(params.UnlockingEpochs))

	testCases := []struct {
		name      string
		input     *types.MsgUnlock
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid unlock",
			input: &types.MsgUnlock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            validCoin,
			},
			expErr: false,
		},
		{
			name: "insufficient lockup",
			input: &types.MsgUnlock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(500)),
			},
			expErr:    true,
			expErrMsg: "subtract lockup",
		},
		{
			name: "invalid stake amount (zero)",
			input: &types.MsgUnlock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            zeroCoin,
			},
			expErr:    true,
			expErrMsg: "invalid amount",
		},
		{
			name: "invalid stake amount (negative)",
			input: &types.MsgUnlock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake: sdk.Coin{
					Denom:  appparams.DefaultBondDenom,
					Amount: negativeAmount,
				},
			},
			expErr:    true,
			expErrMsg: "invalid amount",
		},
		{
			name: "non-existent lockup",
			input: &types.MsgUnlock{
				DelegatorAddress: "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				ValidatorAddress: valAddr,
				Stake:            validCoin,
			},
			expErr:    true,
			expErrMsg: "subtract lockup",
		},
		{
			name: "invalid delegator address",
			input: &types.MsgUnlock{
				DelegatorAddress: "invalid-delegator-address",
				ValidatorAddress: valAddr,
				Stake:            validCoin,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "invalid validator address",
			input: &types.MsgUnlock{
				DelegatorAddress: delAddr,
				ValidatorAddress: "invalid-validator-address",
				Stake:            validCoin,
			},
			expErr:    true,
			expErrMsg: "validator address",
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

			resp, err := ms.Unlock(sdkCtx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				lockup := k.GetLockup(sdkCtx, delAddress, valAddress)
				require.Nil(t, lockup, "Lockup should be nil after valid unlock")

				require.WithinDuration(t, expectedUnlockTime, resp.CompletionTime, time.Second)
			}
		})
	}
}
