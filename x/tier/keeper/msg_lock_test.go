package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func TestMsgLock(t *testing.T) {
	k, ms, ctx := setupMsgServer(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	validCoin1 := sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(100))
	validCoin2 := sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(3000))
	zeroCoin := sdk.NewCoin(appparams.DefaultBondDenom, math.ZeroInt())
	negativeAmount := math.NewInt(-1000)

	delAddr := "source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9"
	valAddr := "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm"

	delAddress, err := sdk.AccAddressFromBech32(delAddr)
	require.NoError(t, err)
	valAddress, err := sdk.ValAddressFromBech32(valAddr)
	require.NoError(t, err)

	initialDelegatorBalance := math.NewInt(2000)
	initializeDelegator(t, &k, sdkCtx, delAddress, initialDelegatorBalance)
	initialValidatorBalance := math.NewInt(1000)
	initializeValidator(t, k.GetStakingKeeper().(*stakingkeeper.Keeper), sdkCtx, valAddress, initialValidatorBalance)

	testCases := []struct {
		name      string
		input     *types.MsgLock
		expErr    bool
		expErrMsg string
	}{
		{
			name: "valid lock",
			input: &types.MsgLock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            validCoin1,
			},
			expErr: false,
		},
		{
			name: "insufficient funds",
			input: &types.MsgLock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            validCoin2,
			},
			expErr:    true,
			expErrMsg: "insufficient funds",
		},
		{
			name: "invalid stake amount (zero)",
			input: &types.MsgLock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            zeroCoin,
			},
			expErr:    true,
			expErrMsg: "invalid amount",
		},
		{
			name: "invalid stake amount (negative)",
			input: &types.MsgLock{
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.Lock(sdkCtx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
