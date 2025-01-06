package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/app/params"
	testutil "github.com/sourcenetwork/sourcehub/testutil"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func Test_MintCredit(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		amt     int64
		wantErr bool
	}{
		{
			name:    "Mint valid credit",
			addr:    "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
			amt:     100,
			wantErr: false,
		},
		{
			name:    "Mint zero credit",
			addr:    "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
			amt:     0,
			wantErr: true,
		},
		{
			name:    "Invalid address",
			addr:    "",
			amt:     100,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := sdk.AccAddress{}
			if tt.addr != "" {
				addr = sdk.MustAccAddressFromBech32(tt.addr)
			}
			amt := math.NewInt(tt.amt)

			k, ctx := testutil.SetupKeeper(t)

			err := k.MintCredit(ctx, addr, amt)
			if (err != nil) != tt.wantErr {
				t.Errorf("MintCredit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_BurnAllCredits(t *testing.T) {
	tests := []struct {
		name           string
		creditBalances map[string]int64
		openBalances   map[string]int64
		wantErr        bool
	}{
		{
			name:           "Burn all credits successfully (single address)",
			creditBalances: map[string]int64{"source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et": 100},
			openBalances:   map[string]int64{},
			wantErr:        false,
		},
		{
			name:           "No addresses have credits",
			creditBalances: map[string]int64{},
			openBalances:   map[string]int64{},
			wantErr:        false,
		},
		{
			name: "Multiple addresses with credits",
			creditBalances: map[string]int64{
				"source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et": 50,
				"source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9": 150,
			},
			openBalances: map[string]int64{},
			wantErr:      false,
		},
		{
			name: "Burn credits when addresses also hold $OPEN",
			creditBalances: map[string]int64{
				"source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et": 80,
				"source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9": 200,
			},
			openBalances: map[string]int64{
				"source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et": 9999,
				"source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy": 888,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, ctx := testutil.SetupKeeper(t)

			// Mint and distribute credits
			for addrStr, balance := range tt.creditBalances {
				addr := sdk.MustAccAddressFromBech32(addrStr)
				coins := sdk.NewCoins(sdk.NewCoin(params.CreditDenom, math.NewInt(balance)))
				err := k.GetBankKeeper().MintCoins(ctx, types.ModuleName, coins)
				require.NoError(t, err, "MintCoins failed")
				err = k.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins)
				require.NoError(t, err, "SendCoinsFromModuleToAccount failed")
			}

			// Mint and distribute $OPEN
			for addrStr, balance := range tt.openBalances {
				addr := sdk.MustAccAddressFromBech32(addrStr)
				coins := sdk.NewCoins(sdk.NewCoin(params.OpenDenom, math.NewInt(balance)))
				err := k.GetBankKeeper().MintCoins(ctx, types.ModuleName, coins)
				require.NoError(t, err, "MintCoins $OPEN failed")
				err = k.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins)
				require.NoError(t, err, "SendCoinsFromModuleToAccount $OPEN failed")
			}

			// Burn all credits
			err := k.BurnAllCredits(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("BurnAllCredits() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify that all credit balances are zero
			for addrStr, origBalance := range tt.creditBalances {
				addr := sdk.MustAccAddressFromBech32(addrStr)
				bal := k.GetBankKeeper().GetBalance(ctx, addr, params.CreditDenom)
				if !bal.IsZero() {
					t.Errorf("Expected all credit burned for %s, original = %d, still found = %s",
						addrStr, origBalance, bal.Amount)
				}
			}

			// Verify that $OPEN balances are unchanged
			for addrStr, expectedBalance := range tt.openBalances {
				addr := sdk.MustAccAddressFromBech32(addrStr)
				bal := k.GetBankKeeper().GetBalance(ctx, addr, params.OpenDenom)
				if !bal.Amount.Equal(math.NewInt(expectedBalance)) {
					t.Errorf("Non-credit denom incorrectly burned. For %s, got = %d, expected = %d",
						addrStr, bal.Amount.Int64(), expectedBalance)
				}
			}
		})
	}
}

func Test_ResetAllCredits(t *testing.T) {
	tests := []struct {
		name           string
		lockups        map[string][]int64
		expectedCredit map[string]int64
		wantErr        bool
	}{
		{
			name: "Reset all credits successfully (single address, single lockup)",
			lockups: map[string][]int64{
				"source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et": {100},
			},
			expectedCredit: map[string]int64{
				"source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et": 100,
			},
			wantErr: false,
		},
		{
			name:           "No lockups",
			lockups:        map[string][]int64{},
			expectedCredit: map[string]int64{},
			wantErr:        false,
		},
		{
			name: "Multiple addresses with multiple lockups",
			lockups: map[string][]int64{
				"source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et": {50, 50},
				"source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9": {10, 20},
				"source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy": {10, 10, 30},
				"source1cy0p47z24ejzvq55pu3lesxwf73xnrnd0lyxme": {},
			},
			expectedCredit: map[string]int64{
				"source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et": 100, // 50 + 50
				"source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9": 30,  // 10 + 20
				"source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy": 50,  // 10 + 10 + 30
			},
			wantErr: false,
		},
		{
			name: "Multiple addresses with multiple lockups (with reward rates)",
			lockups: map[string][]int64{
				"source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et": {100, 100},
				"source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9": {100, 200, 300},
				"source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy": {500, 1000},
				"source1cy0p47z24ejzvq55pu3lesxwf73xnrnd0lyxme": {},
			},
			expectedCredit: map[string]int64{
				"source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et": 210,  // 100 + 100 + (10 rewards)
				"source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9": 780,  // 100 + 200 + 300 + (180 rewards)
				"source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy": 2130, // 500 + 1000 + (630 rewards)
			},
			wantErr: false,
		},
	}

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, ctx := testutil.SetupKeeper(t)

			// Set default params
			err := k.SetParams(ctx, types.DefaultParams())
			require.NoError(t, err)

			// Add lockups
			for addrStr, lockupAmounts := range tt.lockups {
				addr := sdk.MustAccAddressFromBech32(addrStr)
				for _, amt := range lockupAmounts {
					k.AddLockup(ctx, addr, valAddr, math.NewInt(amt))
				}
			}

			// Reset all credits
			err = k.ResetAllCredits(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResetAllCredits() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check expected credits
			for addrStr, expected := range tt.expectedCredit {
				addr := sdk.MustAccAddressFromBech32(addrStr)
				bal := k.GetBankKeeper().GetBalance(ctx, addr, params.CreditDenom)
				if !bal.Amount.Equal(math.NewInt(expected)) {
					t.Errorf("Incorrect credit balance for %s, got = %v, expected = %v",
						addrStr, bal.Amount, expected)
				}
			}

			// Addresses not in expectedCredit should have zero credit
			for addrStr := range tt.lockups {
				if _, ok := tt.expectedCredit[addrStr]; !ok {
					addr := sdk.MustAccAddressFromBech32(addrStr)
					bal := k.GetBankKeeper().GetBalance(ctx, addr, params.CreditDenom)
					if !bal.IsZero() {
						t.Errorf("Address %s was not in expectedCredit, but has credit = %v",
							addrStr, bal.Amount)
					}
				}
			}
		})
	}
}
