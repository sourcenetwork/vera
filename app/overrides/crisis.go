package overrides

import (
	"encoding/json"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/x/crisis"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

// CrisisModuleBasic defines a wrapper of the x/crisis module AppModuleBasic to provide custom default genesis state.
type CrisisModuleBasic struct {
	crisis.AppModuleBasic
}

// DefaultGenesis returns custom x/crisis module genesis state.
func (CrisisModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := crisistypes.DefaultGenesisState()
	genState.ConstantFee = sdk.NewCoin(appparams.DefaultBondDenom, sdkmath.NewInt(1000))

	return cdc.MustMarshalJSON(genState)
}
