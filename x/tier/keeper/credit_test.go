package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/app/params"
	epochstypes "github.com/sourcenetwork/sourcehub/x/epochs/types"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func TestMintCredit(t *testing.T) {
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

			k, ctx := setupKeeper(t)

			err := k.mintCredit(ctx, addr, amt)
			if (err != nil) != tt.wantErr {
				t.Errorf("mintCredit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProratedCredit(t *testing.T) {
	k, ctx := setupKeeper(t)

	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name        string
		locked      int64
		locking     int64
		remainingMs int64
		totalMs     int64
		want        int64
	}{
		{
			name:        "Large amount in the middle of the epoch",
			locked:      0,
			locking:     10_000_000,
			remainingMs: 120_000, // 2 minutes
			totalMs:     300_000, // 5 minutes
			want:        5_999_952,
		},
		{
			name:        "Large amount at the first second of the epoch",
			locked:      0,
			locking:     10_000_000,
			remainingMs: 299_000, // 4 minutes 59 seconds
			totalMs:     300_000, // 5 minutes
			want:        14_949_880,
		},
		{
			name:        "Large amount at the last second of the epoch",
			locked:      0,
			locking:     10_000_000,
			remainingMs: 1000,    // 1 second
			totalMs:     300_000, // 5 minutes
			want:        49_999,
		},
		{
			name:        "At the beginning of an epoch",
			locked:      0,
			locking:     100,
			remainingMs: 300_000, // 5 minutes
			totalMs:     300_000, // 5 minutes
			want:        100,
		},
		{
			name:        "In the middle of an epoch",
			locked:      0,
			locking:     100,
			remainingMs: 120_000, // 2 minutes
			totalMs:     300_000, // 5 minutes
			want:        40,
		},
		{
			name:        "At the end of an epoch",
			locked:      0,
			locking:     100,
			remainingMs: 0,
			totalMs:     300_000, // 5 minutes
			want:        0,
		},
		{
			name:        "Negative locking amount",
			locked:      0,
			locking:     -100,
			remainingMs: 120_000, // 2 minutes
			totalMs:     300_000, // 5 minutes
			want:        0,
		},
		{
			name:        "Zero locking amount",
			locked:      0,
			locking:     0,
			remainingMs: 120_000, // 2 minutes
			totalMs:     300_000, // 5 minutes
			want:        0,
		},
		{
			name:        "Locking amount with 1.1 rate",
			locked:      0,
			locking:     100,
			remainingMs: 240_000, // 4 minutes
			totalMs:     300_000, // 5 minutes
			want:        80,
		},
		{
			name:        "Locking amount with 1.2 rate",
			locked:      0,
			locking:     200,
			remainingMs: 180_000, // 3 minutes
			totalMs:     360_000, // 6 minutes
			want:        105,
		},
		{
			name:        "Locking amount with 1.5 rate",
			locked:      0,
			locking:     300,
			remainingMs: 120_000, // 2 minutes
			totalMs:     300_000, // 5 minutes
			want:        132,
		},
		{
			name:        "Zero epoch duration",
			locked:      0,
			locking:     100,
			remainingMs: 0,
			totalMs:     0,
			want:        0,
		},
		{
			name:        "Short epoch duration",
			locked:      0,
			locking:     100,
			remainingMs: 30_000, // 30 seconds
			totalMs:     60_000, // 1 minute
			want:        50,
		},
		{
			name:        "Long epoch duration",
			locked:      0,
			locking:     100,
			remainingMs: 1_296_000_000, // 15 days
			totalMs:     2_592_000_000, // 30 days
			want:        50,
		},
		{
			name:        "Epoch duration less than sinceCurrentEpoch",
			locked:      0,
			locking:     100,
			remainingMs: 600_000, // 10 minutes
			totalMs:     300_000, // 5 minutes
			want:        100,
		},
		{
			name:        "Small locking amount and short epoch",
			locked:      0,
			locking:     9,
			remainingMs: 3000,   // 3 seconds
			totalMs:     10_000, // 10 seconds
			want:        2,
		},
		{
			name:        "Small locking amount and long epoch",
			locked:      0,
			locking:     11,
			remainingMs: 600_000,   // 10 minutes
			totalMs:     7_200_000, // 2 hours
			want:        0,
		},
		{
			name:        "Large locking amount and short epoch",
			locked:      0,
			locking:     1_000_003,
			remainingMs: 1000, // 1 second
			totalMs:     5000, // 5 seconds
			want:        299_976,
		},
		{
			name:        "Large locking amount and long epoch",
			locked:      0,
			locking:     1_000_003,
			remainingMs: 3_600_000,  // 1 hour
			totalMs:     18_000_000, // 5 hours
			want:        299_976,
		},
		{
			name:        "Locking amount causing uneven division",
			locked:      0,
			locking:     7,
			remainingMs: 10_000, // 10 seconds
			totalMs:     33_000, // 33 seconds
			want:        2,
		},
		{
			name:        "Epoch duration causing uneven division",
			locked:      0,
			locking:     1234,
			remainingMs: 4000, // 4 seconds
			totalMs:     9000, // 9 seconds
			want:        769,
		},
		{
			name:        "Locking amount with 1.0 rate and previously locked amount",
			locked:      40,
			locking:     40,
			remainingMs: 120_000, // 2 minutes
			totalMs:     300_000, // 5 minutes
			want:        16,      // 40 * 1.0 * 2/5
		},
		{
			name:        "Locking amount with 1.1 rate and previously locked amount",
			locked:      100,
			locking:     100,
			remainingMs: 120_000, // 2 minutes
			totalMs:     300_000, // 5 minutes
			want:        44,      // 100 * 1.1 * 2/5
		},
		{
			name:        "Locking amount with 1.5 rate and previously locked amount",
			locked:      300,
			locking:     300,
			remainingMs: 120_000, // 2 minutes
			totalMs:     300_000, // 5 minutes
			want:        180,     // 300 * 1.5 * 2/5
		},
		{
			name:        "Locking amount with 1.2 + 1.1 rate and previously locked amount",
			locked:      150,
			locking:     150,
			remainingMs: 120_000, // 2 minutes
			totalMs:     300_000, // 5 minutes
			want:        70,      // ((100 * 1.2) + (50 * 1.1)) * 2/5
		},
		{
			name:        "Locking amount with 1.5 + 1.2 rate and previously locked amount",
			locked:      200,
			locking:     200,
			remainingMs: 120_000, // 2 minutes
			totalMs:     300_000, // 5 minutes
			want:        108,     // ((100 * 1.5) + (100 * 1.2)) * 2/5
		},
		{
			name:        "Locking amount with 1.5 + 1.2 + 1.1 + 1.0 rate and previously locked amount",
			locked:      50,
			locking:     350,
			remainingMs: 120_000, // 2 minutes
			totalMs:     300_000, // 5 minutes
			want:        172,     // ((100 * 1.5) + (100 * 1.2) + (100 * 1.1) + (50 * 1.0)) * 2/5
		},
		{
			name:        "Locking small amount with large previously locked amount",
			locked:      1_000_000,
			locking:     100,
			remainingMs: 120_000, // 2 minutes
			totalMs:     300_000, // 5 minutes
			want:        60,      // 100 * 1.5 * 2/5
		},
		{
			name:        "Locking large amount with large previously locked amount",
			locked:      1_000_000,
			locking:     10_000_000,
			remainingMs: 120_000,   // 2 minutes
			totalMs:     300_000,   // 5 minutes
			want:        6_000_000, // 10,000,000 * 1.5 * 2/5
		},
	}

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	delAddr := sdk.MustAccAddressFromBech32("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(t, err)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx = ctx.WithBlockTime(baseTime).WithBlockHeight(1)

			epochInfo := epochstypes.EpochInfo{
				Identifier:            types.EpochIdentifier,
				CurrentEpoch:          1,
				CurrentEpochStartTime: baseTime.Add(-time.Duration(tc.remainingMs) * time.Millisecond),
				Duration:              time.Duration(tc.totalMs) * time.Millisecond,
			}
			k.epochsKeeper.SetEpochInfo(ctx, epochInfo)

			k.removeLockup(ctx, delAddr, valAddr)
			if tc.locked > 0 {
				k.AddLockup(ctx, delAddr, valAddr, math.NewInt(tc.locked))
			}
			got := k.proratedCredit(ctx, delAddr, math.NewInt(tc.locking))
			require.Equal(t, tc.want, got.Int64())
		})
	}
}

func TestBurnAllCredits(t *testing.T) {
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
			k, ctx := setupKeeper(t)

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
			err := k.burnAllCredits(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("burnAllCredits() error = %v, wantErr %v", err, tt.wantErr)
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

func TestResetAllCredits(t *testing.T) {
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
			k, ctx := setupKeeper(t)

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
			err = k.resetAllCredits(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("resetAllCredits() error = %v, wantErr %v", err, tt.wantErr)
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
