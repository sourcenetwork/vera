package overrides

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"

	epochsmodule "github.com/sourcenetwork/vera/x/epochs/module"
	epochstypes "github.com/sourcenetwork/vera/x/epochs/types"
	tiertypes "github.com/sourcenetwork/vera/x/tier/types"
)

// EpochsModuleBasic defines a wrapper of the x/epochs module AppModuleBasic to provide custom default genesis state.
type EpochsModuleBasic struct {
	epochsmodule.AppModuleBasic
}

// DefaultGenesis returns custom x/epochs module genesis state.
func (EpochsModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := epochstypes.DefaultGenesis()
	genState.Epochs = []epochstypes.EpochInfo{
		epochstypes.NewGenesisEpochInfo(tiertypes.EpochIdentifier, tiertypes.DefaultEpochDuration),
	}
	return cdc.MustMarshalJSON(genState)
}
