package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	appparams "github.com/sourcenetwork/sourcehub/app/params"
	epochstypes "github.com/sourcenetwork/sourcehub/x/epochs/types"
	"github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

type TestCase struct {
	name           string
	input          *types.MsgCancelUnlocking
	expErr         bool
	expErrMsg      string
	expectedAmount math.Int
}

func runMsgTestCase(t *testing.T, tc TestCase, k keeper.Keeper, ms types.MsgServer, initState func() sdk.Context, delAddress sdk.AccAddress, valAddress sdk.ValAddress) {
	ctx := initState()

	err := tc.input.ValidateBasic()
	if err != nil {
		if tc.expErr {
			require.Contains(t, err.Error(), tc.expErrMsg)
			return
		}
		t.Fatalf("unexpected error in ValidateBasic: %v", err)
	}

	resp, err := ms.CancelUnlocking(ctx, tc.input)

	if tc.expErr {
		require.Error(t, err)
		require.Contains(t, err.Error(), tc.expErrMsg)
	} else {
		require.NoError(t, err)
		require.NotNil(t, resp, "Response should not be nil for valid cancel unlocking")

		lockup := k.GetLockup(ctx, delAddress, valAddress)
		require.NotNil(t, lockup, "Lockup should not be nil after cancel unlocking")
		require.Equal(t, tc.expectedAmount, lockup.Amount, "Lockup amount should match expected after cancel unlocking")
	}
}

func TestMsgCancelUnlocking(t *testing.T) {
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
	validCoinRounded := sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(100).Sub(math.OneInt()))
	partialCancelCoin := sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(50))
	excessCoin := sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(500))
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

	initState := func() sdk.Context {
		ctx, _ := sdkCtx.CacheContext()
		initializeDelegator(t, &k, ctx, delAddress, initialDelegatorBalance)
		initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), ctx, valAddress, initialValidatorBalance)
		err = k.Lock(ctx, delAddress, valAddress, validCoin.Amount)
		require.NoError(t, err)
		_, _, _, err = k.Unlock(ctx, delAddress, valAddress, validCoin.Amount)
		require.NoError(t, err)
		return ctx
	}

	testCases := []TestCase{
		{
			name: "invalid stake amount (zero)",
			input: &types.MsgCancelUnlocking{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				CreationHeight:   1,
				Stake:            zeroCoin,
			},
			expErr:    true,
			expErrMsg: "invalid amount",
		},
		{
			name: "invalid stake amount (negative)",
			input: &types.MsgCancelUnlocking{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				CreationHeight:   1,
				Stake: sdk.Coin{
					Denom:  appparams.DefaultBondDenom,
					Amount: negativeAmount,
				},
			},
			expErr:    true,
			expErrMsg: "invalid amount",
		},
		{
			name: "excess unlocking amount",
			input: &types.MsgCancelUnlocking{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				CreationHeight:   1,
				Stake:            excessCoin,
			},
			expErr:    true,
			expErrMsg: "invalid amount",
		},
		{
			name: "non-existent unlocking",
			input: &types.MsgCancelUnlocking{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				CreationHeight:   100,
				Stake:            validCoin,
			},
			expErr:    true,
			expErrMsg: stakingtypes.ErrNoUnbondingDelegation.Error(),
		},
		{
			name: "invalid delegator address",
			input: &types.MsgCancelUnlocking{
				DelegatorAddress: "invalid-address",
				ValidatorAddress: valAddr,
				CreationHeight:   1,
				Stake:            validCoin,
			},
			expErr:    true,
			expErrMsg: "delegator address",
		},
		{
			name: "invalid validator address",
			input: &types.MsgCancelUnlocking{
				DelegatorAddress: delAddr,
				ValidatorAddress: "invalid-validator-address",
				CreationHeight:   1,
				Stake:            validCoin,
			},
			expErr:    true,
			expErrMsg: "validator address",
		},
		{
			name: "valid cancel unlocking (partial)",
			input: &types.MsgCancelUnlocking{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				CreationHeight:   1,
				Stake:            partialCancelCoin,
			},
			expErr:         false,
			expectedAmount: validCoin.Amount.Sub(partialCancelCoin.Amount),
		},
		{
			name: "valid cancel unlocking (full)",
			input: &types.MsgCancelUnlocking{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				CreationHeight:   1,
				Stake:            validCoinRounded,
			},
			expErr:         false,
			expectedAmount: validCoinRounded.Amount,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			runMsgTestCase(t, tc, k, ms, initState, delAddress, valAddress)
		})
	}
}
