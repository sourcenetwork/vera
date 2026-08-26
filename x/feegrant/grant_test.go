package feegrant_test

import (
	"testing"
	"time"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"
	"github.com/sourcenetwork/vera/x/feegrant"
	"github.com/sourcenetwork/vera/x/feegrant/module"

	codecaddress "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
)

func TestGrant(t *testing.T) {
	addressCodec := codecaddress.NewBech32Codec("source")
	key := storetypes.NewKVStoreKey(feegrant.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModuleBasic{})

	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: time.Now()})

	addr, err := addressCodec.StringToBytes("source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9")
	require.NoError(t, err)
	addr2, err := addressCodec.StringToBytes("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	uopen := sdk.NewCoins(sdk.NewInt64Coin("uopen", 555))
	now := ctx.BlockTime()
	oneYear := now.AddDate(1, 0, 0)

	zeroUopen := sdk.NewCoins(sdk.NewInt64Coin("uopen", 0))

	cases := map[string]struct {
		granter sdk.AccAddress
		grantee sdk.AccAddress
		limit   sdk.Coins
		expires time.Time
		valid   bool
	}{
		"good": {
			granter: addr2,
			grantee: addr,
			limit:   uopen,
			expires: oneYear,
			valid:   true,
		},
		"no grantee": {
			granter: addr2,
			grantee: nil,
			limit:   uopen,
			expires: oneYear,
			valid:   false,
		},
		"no granter": {
			granter: nil,
			grantee: addr,
			limit:   uopen,
			expires: oneYear,
			valid:   false,
		},
		"self-grant": {
			granter: addr2,
			grantee: addr2,
			limit:   uopen,
			expires: oneYear,
			valid:   false,
		},
		"zero allowance": {
			granter: addr2,
			grantee: addr,
			limit:   zeroUopen,
			expires: oneYear,
			valid:   false,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			grant, err := feegrant.NewGrant(tc.granter, tc.grantee, &feegrant.BasicAllowance{
				SpendLimit: tc.limit,
				Expiration: &tc.expires,
			})
			require.NoError(t, err)
			err = grant.ValidateBasic()

			if !tc.valid {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// if it is valid, let's try to serialize, deserialize, and make sure it matches
			bz, err := encCfg.Codec.Marshal(&grant)
			require.NoError(t, err)
			var loaded feegrant.Grant
			err = encCfg.Codec.Unmarshal(bz, &loaded)
			require.NoError(t, err)

			err = loaded.ValidateBasic()
			require.NoError(t, err)

			require.Equal(t, grant, loaded)
		})
	}
}

func TestDIDGrant(t *testing.T) {
	addressCodec := codecaddress.NewBech32Codec("source")
	key := storetypes.NewKVStoreKey(feegrant.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModuleBasic{})

	ctx := testCtx.Ctx.WithBlockHeader(cmtproto.Header{Time: time.Now()})

	addr, err := addressCodec.StringToBytes("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	require.NoError(t, err)
	uopen := sdk.NewCoins(sdk.NewInt64Coin("uopen", 555))
	now := ctx.BlockTime()
	oneYear := now.AddDate(1, 0, 0)

	zeroUopen := sdk.NewCoins(sdk.NewInt64Coin("uopen", 0))

	cases := map[string]struct {
		granter    sdk.AccAddress
		granteeDID string
		limit      sdk.Coins
		expires    time.Time
		valid      bool
	}{
		"good": {
			granter:    addr,
			granteeDID: "did:example:bob",
			limit:      uopen,
			expires:    oneYear,
			valid:      true,
		},
		"empty DID": {
			granter:    addr,
			granteeDID: "",
			limit:      uopen,
			expires:    oneYear,
			valid:      false,
		},
		"invalid DID - no prefix": {
			granter:    addr,
			granteeDID: "example:bob",
			limit:      uopen,
			expires:    oneYear,
			valid:      false,
		},
		"invalid DID - short": {
			granter:    addr,
			granteeDID: "did:",
			limit:      uopen,
			expires:    oneYear,
			valid:      false,
		},
		"no granter": {
			granter:    nil,
			granteeDID: "did:example:bob",
			limit:      uopen,
			expires:    oneYear,
			valid:      false,
		},
		"zero allowance": {
			granter:    addr,
			granteeDID: "did:example:bob",
			limit:      zeroUopen,
			expires:    oneYear,
			valid:      false,
		},
	}

	for name, tc := range cases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			grant, err := feegrant.NewDIDGrant(tc.granter, tc.granteeDID, &feegrant.BasicAllowance{
				SpendLimit: tc.limit,
				Expiration: &tc.expires,
			})

			if !tc.valid {
				// For invalid cases, we expect either NewDIDGrant to fail or ValidateBasic to fail
				if err != nil {
					require.Error(t, err)
					return
				}
				err = grant.ValidateBasic()
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			err = grant.ValidateBasic()
			require.NoError(t, err)

			// if it is valid, let's try to serialize, deserialize, and make sure it matches
			bz, err := encCfg.Codec.Marshal(&grant)
			require.NoError(t, err)
			var loaded feegrant.DIDGrant
			err = encCfg.Codec.Unmarshal(bz, &loaded)
			require.NoError(t, err)

			err = loaded.ValidateBasic()
			require.NoError(t, err)

			require.Equal(t, grant, loaded)
		})
	}
}
