package overrides

import (
	"encoding/json"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"

	"github.com/cosmos/cosmos-sdk/x/mint"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

// MintModuleBasic defines a wrapper of the x/mint module AppModuleBasic to provide custom default genesis state.
type MintModuleBasic struct {
	mint.AppModuleBasic
}

// DefaultGenesis returns custom x/mint module genesis state.
func (MintModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := minttypes.DefaultGenesisState()
	genState.Params.MintDenom = appparams.DefaultBondDenom
	genState.Params.BlocksPerYear = appparams.BlocksPerYear
	genState.Params.InflationMin = math.LegacyMustNewDecFromStr(appparams.InflationMin)
	genState.Params.InflationMax = math.LegacyMustNewDecFromStr(appparams.InflationMax)
	genState.Params.InflationRateChange = math.LegacyMustNewDecFromStr(appparams.InflationRateChange)

	return cdc.MustMarshalJSON(genState)
}
