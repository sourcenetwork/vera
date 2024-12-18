package keeper_test

import (
	"testing"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	testutil "github.com/sourcenetwork/sourcehub/testutil"
)

func Test_MintCredit(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		amt     int64
		wantErr bool
	}{
		{
			name:    "Mint valid credit",
			addr:    "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
			amt:     100,
			wantErr: false,
		},
		{
			name:    "Mint zero credit",
			addr:    "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
			amt:     0,
			wantErr: true,
		},
		{
			name:    "Invalid address",
			addr:    "",
			amt:     100,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := sdk.AccAddress{}
			if tt.addr != "" {
				addr = sdk.MustAccAddressFromBech32(tt.addr)
			}
			amt := math.NewInt(tt.amt)

			k, ctx := testutil.SetupKeeper(t)

			err := k.MintCredit(ctx, addr, amt)
			if (err != nil) != tt.wantErr {
				t.Errorf("MintCredit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
