package tier

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	tierkeeper "github.com/sourcenetwork/sourcehub/x/tier/keeper"
)

var TestDeveloperAddrs = []string{
	"source1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s68fcw9",
	"source18jtkvj0995fy7lggqayg2f5syna92ndq5mkuv4",
	"source1cy0p47z24ejzvq55pu3lesxwf73xnrnd0lyxme",
	"source1wxyxr8h3vag0z825hexskdnc7xe72evulxlplv",
	"source1l7gwya9xrgtf5q58q9aqnahh63v99e54vrxwdg",
	"source1jx8s2hy8g74mvhy09whswxcczh22jus9y5t6vd",
	"source1s266eutknzy6vzt7pmxfukch2x93xevq0xqtce",
	"source1dte8lmpu236zmephrc35v95mhsz7ap9zzrld8a",
	"source1fxt3xht88zmtsdvj7u7y537relhehvn2n3hg6q",
	"source1fheg4eh4l9kkq8gczfx5erkrmunm2dj37yxzpx",
	"source19puvx5eypaft0y2hefj6f6s70thm8tadpzq3fe",
	"source1cjuyh88peyu2jztgw6j5523lq9ex06cgt45jmn",
	"source1rr9p6sck6ejdt07pnn0u5zeamfu3g7tz832czj",
}

var TestUserAddrs = []string{
	"source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et",
	"source1n34fvpteuanu2nx2a4hql4jvcrcnal3gsrjppy",
	"source1fa3jnszqhulqwu5amsel280une8t8r5tn92k0y",
	"source1x404mhhw9rl2vqk54eh8v2lu86qlllg5cldq0y",
	"source1577qwqy8enqhh7lqx87rpk46mnhawlceg4xzl6",
	"source1nqxutf4jvfm54j8jrhmafc62lp2pyzj5djxz8z",
	"source1g65up5twggwcv77cjfem3z2ac602qev9hshcj5",
	"source1vvt30uh7kxcdwcevstvzqkmqmzhvt4js0qv33q",
	"source1tnhv4sm4r42hacn9xqm72w0tqy4jm7k22vkcka",
	"source1l6c49y4z4vg6msuqkr37cvty4xs3gr850ypxju",
	"source1wrf55454lqpctztnt7gkm3k7sa3c06r06natzf",
	"source1kmvg2d4fv56k4jg9s76l0nzpg5psnv4l3qp5kv",
}

const TestValidatorAddr = "sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm"

type TestAddressFactory struct {
	devIdx  int
	userIdx int
}

func NewTestAddressFactory() *TestAddressFactory {
	return &TestAddressFactory{
		devIdx:  0,
		userIdx: 0,
	}
}

func (f *TestAddressFactory) NextUserDid() string {
	did := fmt.Sprintf("did:key:user%d", f.userIdx)
	f.userIdx++
	return did
}

func (f *TestAddressFactory) NextDeveloper(t *testing.T, k *tierkeeper.Keeper, ctx sdk.Context) sdk.AccAddress {
	addr := mustAccFromBech32(TestDeveloperAddrs[f.devIdx%len(TestDeveloperAddrs)])
	f.devIdx++
	keepertest.CreateAccount(t, k, ctx, addr)
	return addr
}

func (f *TestAddressFactory) NextUser(t *testing.T, k *tierkeeper.Keeper, ctx sdk.Context) sdk.AccAddress {
	addr := mustAccFromBech32(TestUserAddrs[f.userIdx%len(TestUserAddrs)])
	f.userIdx++
	keepertest.CreateAccount(t, k, ctx, addr)
	return addr
}

func (f *TestAddressFactory) NextPair(t *testing.T, k *tierkeeper.Keeper, ctx sdk.Context) (sdk.AccAddress, sdk.AccAddress) {
	return f.NextDeveloper(t, k, ctx), f.NextUser(t, k, ctx)
}

func mustAccFromBech32(s string) sdk.AccAddress {
	addr, err := sdk.AccAddressFromBech32(s)
	if err != nil {
		panic(err)
	}
	return addr
}
