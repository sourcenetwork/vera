package overrides

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmos/cosmos-sdk/x/bank"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

// BankModuleBasic defines a wrapper of the x/bank module AppModuleBasic to provide custom default genesis state.
type BankModuleBasic struct {
	bank.AppModuleBasic
}

// DefaultGenesis returns custom x/bank module genesis state.
func (BankModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	openMetadata := banktypes.Metadata{
		Description: "OPEN is the native staking token of SourceHub",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    appparams.MicroOpenDenom,
				Exponent: 0,
				Aliases:  []string{appparams.OpenDenom},
			},
			{
				Denom:    appparams.OpenDenom,
				Exponent: 6,
			},
		},
		Base:    appparams.MicroOpenDenom,
		Display: appparams.OpenDenom,
		Name:    appparams.OpenName,
		Symbol:  appparams.OpenSymbol,
	}

	creditMetadata := banktypes.Metadata{
		Description: "Credit is the utility token for access services on SourceHub. Non transferrable.",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    appparams.MicroCreditDenom,
				Exponent: 0,
				Aliases:  []string{appparams.CreditDenom},
			},
			{
				Denom:    appparams.CreditDenom,
				Exponent: 6,
			},
		},
		Base:    appparams.MicroCreditDenom,
		Display: appparams.CreditDenom,
		Name:    appparams.CreditName,
		Symbol:  appparams.CreditSymbol,
	}

	creditSendEnabled := banktypes.SendEnabled{
		Denom:   appparams.MicroCreditDenom,
		Enabled: false,
	}

	genState := banktypes.DefaultGenesisState()
	genState.DenomMetadata = append(genState.DenomMetadata, openMetadata)
	genState.DenomMetadata = append(genState.DenomMetadata, creditMetadata)
	genState.SendEnabled = append(genState.SendEnabled, creditSendEnabled)

	return cdc.MustMarshalJSON(genState)
}
