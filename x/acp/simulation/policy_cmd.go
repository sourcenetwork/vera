package simulation

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/sourcenetwork/vera/x/acp/keeper"
	"github.com/sourcenetwork/vera/x/acp/types"
)

func SimulateMsgPolicyCmd(
	ak types.AccountKeeper,
	bk types.BankKeeper,
	k *keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgSignedPolicyCmd{
			Creator: simAccount.Address.String(),
		}

		// TODO: Handling the PolicyCmd simulation

		return simtypes.NoOpMsg(types.ModuleName, sdk.MsgTypeURL(msg), "PolicyCmd simulation not implemented"), nil, nil
	}
}
