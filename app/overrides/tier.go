package overrides

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"

	tiermodule "github.com/sourcenetwork/sourcehub/x/tier/module"
	tiertypes "github.com/sourcenetwork/sourcehub/x/tier/types"
)

// TierModuleBasic defines a wrapper of the x/tier module AppModuleBasic to provide custom default genesis state.
type TierModuleBasic struct {
	tiermodule.AppModuleBasic
}

// DefaultGenesis returns custom x/tier module genesis state.
func (TierModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := tiertypes.DefaultGenesis()
	return cdc.MustMarshalJSON(genState)
}
