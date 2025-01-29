package params

import (
	"context"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

const (
	OpenDenom   = "open"
	CreditDenom = "credit"

	DefaultBondDenom = OpenDenom
)

var denomMetadatas = []banktypes.Metadata{
	{
		Description: "OPEN is the native staking token of SourceHub",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    "open",
				Exponent: 0,
			},
			{
				Denom:    "uopen",
				Exponent: 6,
			},
		},
		Base:    "open",
		Display: "open",
		Name:    "Source Open",
		Symbol:  "OPEN",
	},
	{
		Description: "Credit is the utility token for access services on SourceHub",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    "credit",
				Exponent: 0,
			},
			{
				Denom:    "ucredit",
				Exponent: 6,
			},
		},
		Base:    "credit",
		Display: "credit",
		Name:    "Source Credit",
		Symbol:  "CREDIT",
	},
}

// RegisterDenoms registers token denoms.
func RegisterDenoms(ctx context.Context, bk bankkeeper.Keeper) {
	for _, denomMetadata := range denomMetadatas {
		bk.SetDenomMetaData(ctx, denomMetadata)
	}
}
