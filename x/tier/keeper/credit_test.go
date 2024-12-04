package keeper_test

import (
	"fmt"
	"reflect"
	"testing"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	testutil "github.com/sourcenetwork/sourcehub/testutil"
	"github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

func Test_CalculateCredit(t *testing.T) {
	rateList := []types.Rate{
		{Amount: math.NewInt(300), Rate: 150},
		{Amount: math.NewInt(200), Rate: 120},
		{Amount: math.NewInt(100), Rate: 110},
		{Amount: math.NewInt(0), Rate: 100},
	}

	tests := []struct {
		lockedAmt  int64
		lockingAmt int64
		want       int64
	}{
		{
			lockedAmt:  100,
			lockingAmt: 0,
			want:       0,
		},
		{
			lockedAmt:  250,
			lockingAmt: 0,
			want:       0,
		},
		{
			lockedAmt:  0,
			lockingAmt: 100,
			want:       100,
		},
		{
			lockedAmt:  0,
			lockingAmt: 200,
			want:       (100 * 1.0) + (100 * 1.1),
		},
		{
			lockedAmt:  0,
			lockingAmt: 250,
			want:       (100 * 1.0) + (100 * 1.1) + (50 * 1.2),
		},
		{
			lockedAmt:  0,
			lockingAmt: 300,
			want:       (100 * 1.0) + (100 * 1.1) + (100 * 1.2),
		},
		{
			lockedAmt:  0,
			lockingAmt: 350,
			want:       (100 * 1.0) + (100 * 1.1) + (100 * 1.2) + (50 * 1.5),
		},
		{
			lockedAmt:  0,
			lockingAmt: 600,
			want:       (100 * 1.0) + (100 * 1.1) + (100 * 1.2) + (300 * 1.5),
		},
		{
			lockedAmt:  100,
			lockingAmt: 100,
			want:       (100 * 1.1),
		},
		{
			lockedAmt:  200,
			lockingAmt: 100,
			want:       (100 * 1.2),
		},
		{
			lockedAmt:  150,
			lockingAmt: 150,
			want:       (50 * 1.1) + (100 * 1.2),
		},
		{
			lockedAmt:  50,
			lockingAmt: 400,
			want:       (50 * 1.0) + (100 * 1.1) + (100 * 1.2) + (150 * 1.5),
		},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("%d adds %d", tt.lockedAmt, tt.lockingAmt)
		oldLock := math.NewInt(tt.lockedAmt)
		newLock := math.NewInt(tt.lockingAmt)
		want := math.NewInt(tt.want)

		t.Run(name, func(t *testing.T) {
			if got := keeper.CalculateCredit(rateList, oldLock, newLock); !reflect.DeepEqual(got, want) {
				t.Errorf("CalculateCredit() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
