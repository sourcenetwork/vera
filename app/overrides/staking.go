package overrides

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"

	staking "github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

// StakingModuleBasic defines a wrapper of the x/staking module AppModuleBasic to provide custom default genesis state.
type StakingModuleBasic struct {
	staking.AppModuleBasic
}

// DefaultGenesis returns custom x/staking module genesis state.
func (StakingModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := stakingtypes.DefaultGenesisState()
	genState.Params.BondDenom = appparams.DefaultBondDenom

	return cdc.MustMarshalJSON(genState)
}
