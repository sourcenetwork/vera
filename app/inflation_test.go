package app

import (
	"testing"

	"cosmossdk.io/math"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	"github.com/stretchr/testify/require"
)

func TestCalculateInflation(t *testing.T) {
	minter := minttypes.Minter{
		Inflation: math.LegacyMustNewDecFromStr(appparams.InitialInflation),
	}

	params := minttypes.DefaultParams()
	params.GoalBonded = math.LegacyMustNewDecFromStr(appparams.GoalBonded)
	params.InflationMin = math.LegacyMustNewDecFromStr(appparams.InflationMin)
	params.InflationMax = math.LegacyMustNewDecFromStr(appparams.InflationMax)
	params.InflationRateChange = math.LegacyMustNewDecFromStr(appparams.InflationRateChange)

	tests := []struct {
		name                     string
		devStake                 math.Int
		totalBonded              math.Int
		totalSupply              math.Int
		expectDefaultBondedRatio bool
	}{
		{
			name:                     "zero total supply",
			devStake:                 math.NewInt(100),
			totalBonded:              math.NewInt(1000),
			totalSupply:              math.ZeroInt(),
			expectDefaultBondedRatio: true,
		},
		{
			name:                     "zero total bonded",
			devStake:                 math.NewInt(100),
			totalBonded:              math.ZeroInt(),
			totalSupply:              math.NewInt(5000),
			expectDefaultBondedRatio: true,
		},
		{
			name:                     "totalBonded < devStake",
			devStake:                 math.NewInt(1000),
			totalBonded:              math.NewInt(500),
			totalSupply:              math.NewInt(5000),
			expectDefaultBondedRatio: true,
		},
		{
			name:                     "totalBonded > totalSupply",
			devStake:                 math.NewInt(1000),
			totalBonded:              math.NewInt(6000),
			totalSupply:              math.NewInt(5000),
			expectDefaultBondedRatio: true,
		},
		{
			name:                     "valid inputs (bondedRatio < GoalBonded)",
			devStake:                 math.NewInt(100),
			totalBonded:              math.NewInt(1000),
			totalSupply:              math.NewInt(5000),
			expectDefaultBondedRatio: false,
		},
		{
			name:                     "valid inputs (bondedRatio > GoalBonded)",
			devStake:                 math.NewInt(100),
			totalBonded:              math.NewInt(4000),
			totalSupply:              math.NewInt(5000),
			expectDefaultBondedRatio: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var defaultBondedRatio, adjustedBondedRatio math.LegacyDec

			if tc.totalSupply.IsZero() {
				defaultBondedRatio = math.LegacyZeroDec()
				adjustedBondedRatio = math.LegacyZeroDec()
			} else {
				defaultBondedRatio = tc.totalBonded.ToLegacyDec().QuoInt(tc.totalSupply)
				adjustedBondedRatio = tc.totalBonded.Sub(tc.devStake).ToLegacyDec().QuoInt(tc.totalSupply)
			}

			expectedInflation := minter.NextInflationRate(params, adjustedBondedRatio)
			if tc.expectDefaultBondedRatio {
				expectedInflation = minter.NextInflationRate(params, defaultBondedRatio)
			}

			actualInflation := CalculateInflation(minter, params, defaultBondedRatio, tc.devStake, tc.totalBonded, tc.totalSupply)

			require.Equal(t, expectedInflation, actualInflation, "Test failed: %s", tc.name)
		})
	}
}
