package tier

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	test "github.com/sourcenetwork/sourcehub/testutil"
	"github.com/stretchr/testify/suite"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"
)

// testDelegation is a minimal implementation of stakingtypes.DelegationI.
type testDelegation struct {
	validatorAddr string
}

func (td testDelegation) GetValidatorAddr() string { return td.validatorAddr }
func (td testDelegation) GetDelegatorAddr() string {
	return "source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et"
}
func (td testDelegation) GetShares() math.LegacyDec { return math.LegacyOneDec() }
func (td testDelegation) GetBondedTokens() math.Int { return math.ZeroInt() }
func (td testDelegation) IsBonded() bool            { return false }

type ModuleTestSuite struct {
	suite.Suite

	tierKeeper       keeper.Keeper
	epochsKeeper     *test.MockEpochsKeeper
	bankKeeper       *test.MockBankKeeper
	distrKeeper      *test.MockDistributionKeeper
	stakingKeeper    *test.MockStakingKeeper
	encCfg           test.EncodingConfig
	ctx              sdk.Context
	msgServer        types.MsgServer
	key              *storetypes.KVStoreKey
	authorityAccount sdk.AccAddress
	delAddr          sdk.AccAddress
	valAddr          sdk.ValAddress
}

func TestModuleTestSuite(t *testing.T) {
	suite.Run(t, new(ModuleTestSuite))
}

func (suite *ModuleTestSuite) SetupTest() {
	suite.encCfg = test.CreateTestEncodingConfig()
	suite.key = storetypes.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), suite.key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx

	ctrl := gomock.NewController(suite.T())

	suite.bankKeeper = test.NewMockBankKeeper(ctrl)
	suite.distrKeeper = test.NewMockDistributionKeeper(ctrl)
	suite.stakingKeeper = test.NewMockStakingKeeper(ctrl)
	suite.epochsKeeper = test.NewMockEpochsKeeper(ctrl)
	suite.authorityAccount = sdk.AccAddress([]byte("authority"))

	suite.tierKeeper = keeper.NewKeeper(
		suite.encCfg.Codec,
		runtime.NewKVStoreService(suite.key),
		log.NewNopLogger(),
		suite.authorityAccount.String(),
		suite.bankKeeper,
		suite.stakingKeeper,
		suite.epochsKeeper,
		suite.distrKeeper,
	)

	err := suite.tierKeeper.SetParams(suite.ctx, types.DefaultParams())
	suite.Require().NoError(err)

	suite.msgServer = keeper.NewMsgServerImpl(suite.tierKeeper)

	suite.delAddr, _ = sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	suite.valAddr, _ = sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
}

func (suite *ModuleTestSuite) TestBeginBlock() {
	tierModuleAddr := authtypes.NewModuleAddress(types.ModuleName)
	delegation := testDelegation{validatorAddr: suite.valAddr.String()}
	totalReward := math.NewInt(100)
	rewardCoins := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, totalReward))

	testCases := []struct {
		name                       string
		expectedInsurancePool      math.Int
		expectedTimesSentToInsPool int
	}{
		{
			name:                       "Insurance pool below threshold",
			expectedInsurancePool:      math.NewInt(1),
			expectedTimesSentToInsPool: 1,
		},
		{
			name:                       "Insurance pool is full",
			expectedInsurancePool:      math.NewInt(100_000_000_000),
			expectedTimesSentToInsPool: 0,
		},
	}

	suite.stakingKeeper.
		EXPECT().
		IterateDelegations(suite.ctx, tierModuleAddr, gomock.Any()).
		DoAndReturn(func(ctx sdk.Context, delegator sdk.AccAddress, fn func(int64, stakingtypes.DelegationI) bool) error {
			fn(0, delegation)
			return nil
		}).Times(len(testCases))

	suite.distrKeeper.
		EXPECT().
		WithdrawDelegationRewards(suite.ctx, tierModuleAddr, suite.valAddr).
		Return(rewardCoins, nil).Times(len(testCases))

	suite.bankKeeper.
		EXPECT().
		SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, types.DeveloperPoolName, gomock.Any()).
		Return(nil).Times(len(testCases))

	suite.bankKeeper.
		EXPECT().
		BurnCoins(gomock.Any(), types.ModuleName, gomock.Any()).
		Return(nil).Times(len(testCases))

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.bankKeeper.
				EXPECT().
				GetBalance(gomock.Any(), authtypes.NewModuleAddress(types.InsurancePoolName), appparams.DefaultBondDenom).
				Return(sdk.NewCoin(appparams.DefaultBondDenom, tc.expectedInsurancePool)).
				Times(1)

			suite.bankKeeper.
				EXPECT().
				SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, types.InsurancePoolName, gomock.Any()).
				Return(nil).Times(tc.expectedTimesSentToInsPool)

			module := NewAppModule(suite.encCfg.Codec, suite.tierKeeper, suite.bankKeeper)
			err := module.BeginBlock(suite.ctx)
			suite.Require().NoError(err)
		})
	}
}
