package tier

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	keepertest "github.com/sourcenetwork/vera/testutil/keeper"
	tierkeeper "github.com/sourcenetwork/vera/x/tier/keeper"
)

var TestDeveloperAddrs = []string{
	"vera1m4f5a896t7fzd9vc7pfgmc3fxkj8n24s2dq2wz",
	"vera18jtkvj0995fy7lggqayg2f5syna92ndqy3lwvj",
	"vera1cy0p47z24ejzvq55pu3lesxwf73xnrndl4d5m7",
	"vera1wxyxr8h3vag0z825hexskdnc7xe72evu0vknlt",
	"vera1l7gwya9xrgtf5q58q9aqnahh63v99e54uf0ud0",
	"vera1jx8s2hy8g74mvhy09whswxcczh22jus957zgv2",
	"vera1s266eutknzy6vzt7pmxfukch2x93xevqlvfec7",
	"vera1dte8lmpu236zmephrc35v95mhsz7ap9zjfkl86",
	"vera1fxt3xht88zmtsdvj7u7y537relhehvn2rm7668",
	"vera1fheg4eh4l9kkq8gczfx5erkrmunm2dj3ww0spp",
	"vera19puvx5eypaft0y2hefj6f6s70thm8tad3gfrf7",
	"vera1cjuyh88peyu2jztgw6j5523lq9ex06cgmlaqm5",
	"vera1rr9p6sck6ejdt07pnn0u5zeamfu3g7tzhmr2z4",
}

var TestUserAddrs = []string{
	"vera1wjj5v5rlf57kayyeskncpu4hwev25ty697gcev",
	"vera1n34fvpteuanu2nx2a4hql4jvcrcnal3gqfmnpr",
	"vera1fa3jnszqhulqwu5amsel280une8t8r5tr0ry0r",
	"vera1x404mhhw9rl2vqk54eh8v2lu86qlllg5g4yj0r",
	"vera1577qwqy8enqhh7lqx87rpk46mnhawlcecl0sla",
	"vera1nqxutf4jvfm54j8jrhmafc62lp2pyzj5ac0s89",
	"vera1g65up5twggwcv77cjfem3z2ac602qev98672jn",
	"vera1vvt30uh7kxcdwcevstvzqkmqmzhvt4jsl29r38",
	"vera1tnhv4sm4r42hacn9xqm72w0tqy4jm7k26xl2k6",
	"vera1l6c49y4z4vg6msuqkr37cvty4xs3gr85lwg5jm",
	"vera1wrf55454lqpctztnt7gkm3k7sa3c06r02e5ezw",
	"vera1kmvg2d4fv56k4jg9s76l0nzpg5psnv4lp2gxkt",
}

const TestValidatorAddr = "veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0"

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
