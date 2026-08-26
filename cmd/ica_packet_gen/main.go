package main

import (
	"encoding/json"
	"flag"
	"fmt"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	icatypes "github.com/cosmos/ibc-go/v10/modules/apps/27-interchain-accounts/types"
	coretypes "github.com/sourcenetwork/acp_core/pkg/types"
	acptypes "github.com/sourcenetwork/vera/x/acp/types"
)

func main() {
	var creator, policy string
	flag.StringVar(&creator, "creator", "", "Host chain ICA address to use as creator")
	flag.StringVar(&policy, "policy", "name: ica test policy", "Policy content")
	flag.Parse()

	if creator == "" {
		panic("--creator is required")
	}

	msg := &acptypes.MsgCreatePolicy{
		Creator:     creator,
		Policy:      policy,
		MarshalType: coretypes.PolicyMarshalingType_YAML,
	}

	anyMsg, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		panic(err)
	}

	cosmosTx := &icatypes.CosmosTx{Messages: []*codectypes.Any{anyMsg}}
	bz, err := gogoproto.Marshal(cosmosTx)
	if err != nil {
		panic(err)
	}

	out := icatypes.InterchainAccountPacketData{
		Type: icatypes.EXECUTE_TX,
		Data: bz,
		Memo: "",
	}

	enc, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		panic(err)
	}

	fmt.Println(string(enc))
}
