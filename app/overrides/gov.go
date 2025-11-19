package overrides

import (
	"encoding/json"
	"time"

	math "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	gov "github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

// GovModuleBasic defines a wrapper of the x/gov module AppModuleBasic to provide custom default genesis state.
type GovModuleBasic struct {
	gov.AppModuleBasic
}

func NewGovModuleBasic() GovModuleBasic {
	return GovModuleBasic{
		AppModuleBasic: gov.NewAppModuleBasic(proposalHandlers()),
	}
}

// proposalHandlers returns supported governance proposal handlers.
func proposalHandlers() []govclient.ProposalHandler {
	return []govclient.ProposalHandler{
		paramsclient.ProposalHandler,
	}
}

// DefaultGenesis returns custom x/gov module genesis state.
func (GovModuleBasic) DefaultGenesis(cdc codec.JSONCodec) json.RawMessage {
	genState := govtypes.DefaultGenesisState()
	oneWeek := time.Duration(time.Hour*24) * 7
	twoWeeks := time.Duration(time.Hour*24) * 14

	genState.Params.MaxDepositPeriod = &twoWeeks
	genState.Params.VotingPeriod = &twoWeeks
	genState.Params.ExpeditedVotingPeriod = &oneWeek

	genState.Params.MinDeposit = sdk.NewCoins(
		sdk.NewCoin(appparams.MicroOpenDenom, math.NewInt(1_000_000_000)),
	)

	genState.Params.ExpeditedMinDeposit = sdk.NewCoins(
		sdk.NewCoin(appparams.MicroOpenDenom, math.NewInt(5_000_000_000)),
	)

	return cdc.MustMarshalJSON(genState)
}
