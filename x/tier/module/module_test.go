package tier

import (
	"testing"
	"time"

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
	epochstypes "github.com/sourcenetwork/sourcehub/x/epochs/types"
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

type KeeperTestSuite struct {
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

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) SetupTest() {
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

	amount := math.NewInt(1_000_000_000_000) // 1m open
	coins := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, amount))
	creditCoins := sdk.NewCoins(sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(249_999_999_980)))

	validator := stakingtypes.Validator{
		OperatorAddress: suite.valAddr.String(),
		Status:          stakingtypes.Bonded,
	}

	epochInfo := epochstypes.EpochInfo{
		Identifier:            types.EpochIdentifier,
		CurrentEpochStartTime: suite.ctx.BlockTime().Add(-10 * time.Minute),
		Duration:              time.Hour,
	}

	suite.bankKeeper.EXPECT().
		MintCoins(gomock.Any(), types.ModuleName, creditCoins).
		Return(nil).Times(1)

	suite.bankKeeper.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), types.ModuleName, suite.delAddr, creditCoins).
		Return(nil).Times(1)

	suite.stakingKeeper.EXPECT().
		GetValidator(gomock.Any(), suite.valAddr).
		Return(validator, nil).Times(1)

	suite.bankKeeper.EXPECT().
		DelegateCoinsFromAccountToModule(gomock.Any(), suite.delAddr, types.ModuleName, coins).
		Return(nil).Times(1)

	suite.stakingKeeper.EXPECT().
		Delegate(gomock.Any(), gomock.Any(), amount, stakingtypes.Unbonded, validator, true).
		Return(math.LegacyNewDecFromInt(amount), nil).Times(1)

	suite.epochsKeeper.EXPECT().
		GetEpochInfo(gomock.Any(), types.EpochIdentifier).
		Return(epochInfo).Times(1)

	// Add a lockup before testing begin block, so that there are funds to burn and send to pools
	err = suite.tierKeeper.Lock(suite.ctx, suite.delAddr, suite.valAddr, amount)
	suite.Require().NoError(err)
}

func (suite *KeeperTestSuite) TestBeginBlock() {
	tierModuleAddr := authtypes.NewModuleAddress(types.ModuleName)
	delegation := testDelegation{validatorAddr: suite.valAddr.String()}
	totalReward := math.NewInt(100)
	rewardCoins := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, totalReward))

	testCases := []struct {
		name                       string
		expectedDevPool            math.Int
		expectedInsurancePool      math.Int
		expectedBurn               math.Int
		expectedTimesSentToInsPool int
	}{
		{
			name:                       "Insurance pool below threshold",
			expectedDevPool:            math.NewInt(2),
			expectedInsurancePool:      math.NewInt(1),
			expectedBurn:               math.NewInt(97),
			expectedTimesSentToInsPool: 1,
		},
		{
			name:                       "Insurance pool is full",
			expectedDevPool:            math.NewInt(3),
			expectedInsurancePool:      math.NewInt(100_000_000_000),
			expectedBurn:               math.NewInt(97),
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
				Times(2)

			suite.bankKeeper.
				EXPECT().
				SendCoinsFromModuleToModule(gomock.Any(), types.ModuleName, types.InsurancePoolName, gomock.Any()).
				Return(nil).Times(tc.expectedTimesSentToInsPool)

			module := NewAppModule(suite.encCfg.Codec, suite.tierKeeper, suite.bankKeeper)
			err := module.BeginBlock(suite.ctx)
			suite.Require().NoError(err)

			suite.bankKeeper.
				EXPECT().
				GetBalance(gomock.Any(), authtypes.NewModuleAddress(types.DeveloperPoolName), appparams.DefaultBondDenom).
				Return(sdk.NewCoin(appparams.DefaultBondDenom, tc.expectedDevPool)).
				Times(1)

			developerPoolAddr := authtypes.NewModuleAddress(types.DeveloperPoolName)
			devBalance := suite.bankKeeper.GetBalance(suite.ctx, developerPoolAddr, appparams.DefaultBondDenom)
			suite.Require().True(devBalance.Amount.Equal(tc.expectedDevPool),
				"dev pool balance: expected %s, got %s", tc.expectedDevPool, devBalance.Amount)

			insurancePoolAddr := authtypes.NewModuleAddress(types.InsurancePoolName)
			insBalance := suite.bankKeeper.GetBalance(suite.ctx, insurancePoolAddr, appparams.DefaultBondDenom)
			suite.Require().True(insBalance.Amount.Equal(tc.expectedInsurancePool),
				"insurance pool balance: expected %s, got %s", tc.expectedInsurancePool, insBalance.Amount)
		})
	}
}
