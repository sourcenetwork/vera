package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	epochstypes "github.com/sourcenetwork/sourcehub/x/epochs/types"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func TestMsgRedelegate(t *testing.T) {
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

	validCoin := sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(100))
	zeroCoin := sdk.NewCoin(appparams.DefaultBondDenom, math.ZeroInt())
	negativeAmount := math.NewInt(-100)
	initialDelegatorBalance := math.NewInt(2000)
	initialSrcValidatorBalance := math.NewInt(1000)
	initialDstValidatorBalance := math.NewInt(500)

	delAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	srcValAddr := "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm"
	dstValAddr := "sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f"

	delAddress, err := sdk.AccAddressFromBech32(delAddr)
	require.NoError(t, err)
	srcValAddress, err := sdk.ValAddressFromBech32(srcValAddr)
	require.NoError(t, err)
	dstValAddress, err := sdk.ValAddressFromBech32(dstValAddr)
	require.NoError(t, err)

	initializeDelegator(t, &k, sdkCtx, delAddress, initialDelegatorBalance)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, srcValAddress, initialSrcValidatorBalance)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, dstValAddress, initialDstValidatorBalance)

	// lock some tokens to test redelegate logic
	err = k.Lock(ctx, delAddress, srcValAddress, validCoin.Amount)
	require.NoError(t, err)

	stakingParams, err := k.GetStakingKeeper().(*stakingkeeper.Keeper).GetParams(ctx)
	require.NoError(t, err)

	// expectedCompletionTime should match the default staking unbonding time (e.g. 21 days)
	expectedCompletionTime := sdkCtx.BlockTime().Add(stakingParams.UnbondingTime)

	testCases := []struct {
		name      string
		input     *types.MsgRedelegate
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid redelegate",
			input: &types.MsgRedelegate{
				DelegatorAddress:    delAddr,
				SrcValidatorAddress: srcValAddr,
				DstValidatorAddress: dstValAddr,
				Stake:               validCoin,
			},
			expErr: false,
		},
		{
			name: "insufficient lockup",
			input: &types.MsgRedelegate{
				DelegatorAddress:    delAddr,
				SrcValidatorAddress: srcValAddr,
				DstValidatorAddress: dstValAddr,
				Stake:               sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(500)),
			},
			expErr:    true,
			expErrMsg: "subtract lockup from source validator",
		},
		{
			name: "invalid stake amount (zero)",
			input: &types.MsgRedelegate{
				DelegatorAddress:    delAddr,
				SrcValidatorAddress: srcValAddr,
				DstValidatorAddress: dstValAddr,
				Stake:               zeroCoin,
			},
			expErr:    true,
			expErrMsg: "invalid amount",
		},
		{
			name: "invalid stake amount (negative)",
			input: &types.MsgRedelegate{
				DelegatorAddress:    delAddr,
				SrcValidatorAddress: srcValAddr,
				DstValidatorAddress: dstValAddr,
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
			input: &types.MsgRedelegate{
				DelegatorAddress:    "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
				SrcValidatorAddress: srcValAddr,
				DstValidatorAddress: dstValAddr,
				Stake:               validCoin,
			},
			expErr:    true,
			expErrMsg: "subtract lockup from source validator",
		},
		{
			name: "source and destination validator are the same",
			input: &types.MsgRedelegate{
				DelegatorAddress:    delAddr,
				SrcValidatorAddress: srcValAddr,
				DstValidatorAddress: srcValAddr,
				Stake:               validCoin,
			},
			expErr:    true,
			expErrMsg: "src and dst validator addresses are the same",
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

			resp, err := ms.Redelegate(sdkCtx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)

				srcLockup := k.GetLockupAmount(sdkCtx, delAddress, srcValAddress)
				require.Equal(t, math.ZeroInt(), srcLockup, "Source validator lockup should be zero after valid redelegate")

				dstLockup := k.GetLockupAmount(sdkCtx, delAddress, dstValAddress)
				require.Equal(t, validCoin.Amount, dstLockup, "Destination validator lockup should equal redelegated amount")

				require.WithinDuration(t, expectedCompletionTime, resp.CompletionTime, time.Second)
			}
		})
	}
}
