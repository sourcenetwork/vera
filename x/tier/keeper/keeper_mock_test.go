package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/golang/mock/gomock"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	test "github.com/sourcenetwork/sourcehub/testutil"
	epochstypes "github.com/sourcenetwork/sourcehub/x/epochs/types"
	"github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/sourcenetwork/sourcehub/x/tier/types"

	"github.com/stretchr/testify/suite"
)

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
}

// TestLock is using mock keepers to verify that required function calls are made as expected on Lock().
func (suite *KeeperTestSuite) TestLock() {
	amount := math.NewInt(1000)
	moduleName := types.ModuleName
	coins := sdk.NewCoins(sdk.NewCoin(appparams.DefaultBondDenom, amount))
	creditCoins := sdk.NewCoins(sdk.NewCoin("credit", math.NewInt(250)))

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	suite.Require().NoError(err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	suite.Require().NoError(err)

	validator := stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
		Status:          stakingtypes.Bonded,
	}

	epochInfo := epochstypes.EpochInfo{
		Identifier:            types.EpochIdentifier,
		CurrentEpochStartTime: suite.ctx.BlockTime().Add(-10 * time.Minute),
		Duration:              time.Hour,
	}

	// confirm that keeper methods are called as expected
	suite.bankKeeper.EXPECT().
		MintCoins(gomock.Any(), types.ModuleName, creditCoins).
		Return(nil).Times(1)

	suite.bankKeeper.EXPECT().
		SendCoinsFromModuleToAccount(gomock.Any(), moduleName, delAddr, creditCoins).
		Return(nil).Times(1)

	suite.stakingKeeper.EXPECT().
		GetValidator(gomock.Any(), valAddr).
		Return(validator, nil).Times(1)

	suite.bankKeeper.EXPECT().
		DelegateCoinsFromAccountToModule(gomock.Any(), delAddr, types.ModuleName, coins).
		Return(nil).Times(1)

	suite.stakingKeeper.EXPECT().
		Delegate(gomock.Any(), gomock.Any(), amount, stakingtypes.Unbonded, validator, true).
		Return(math.LegacyNewDecFromInt(amount), nil).Times(1)

	suite.epochsKeeper.EXPECT().
		GetEpochInfo(gomock.Any(), types.EpochIdentifier).
		Return(epochInfo).Times(1)

	// perform lock and verify that lockup is set correctly
	err = suite.tierKeeper.Lock(suite.ctx, delAddr, valAddr, amount)
	suite.Require().NoError(err)

	lockedAmt := suite.tierKeeper.GetLockupAmount(suite.ctx, delAddr, valAddr)
	suite.Require().Equal(amount, lockedAmt)
}

// TestUnlock is using mock keepers to verify that required function calls are made as expected on Unlock().
func (suite *KeeperTestSuite) TestUnlock() {
	amount := math.NewInt(1000)
	moduleName := types.ModuleName
	unlockingEpochs := int64(2)
	epochDuration := time.Hour

	params := types.Params{
		UnlockingEpochs: unlockingEpochs,
		EpochDuration:   &epochDuration,
	}

	expectedCompletionTime := suite.ctx.BlockTime().Add(time.Hour * 24 * 21)
	expectedUnlockTime := suite.ctx.BlockTime().Add(time.Duration(params.UnlockingEpochs) * *params.EpochDuration)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	suite.Require().NoError(err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	suite.Require().NoError(err)

	// confirm that keeper methods are called as expected

	suite.bankKeeper.EXPECT().
		GetBalance(
			gomock.Any(),
			authtypes.NewModuleAddress(moduleName),
			appparams.DefaultBondDenom,
		).Return(
		sdk.NewCoin(appparams.DefaultBondDenom, math.NewInt(2000)),
	).AnyTimes()

	suite.stakingKeeper.EXPECT().
		ValidateUnbondAmount(
			gomock.Any(),
			authtypes.NewModuleAddress(moduleName),
			valAddr,
			amount,
		).Return(math.LegacyNewDecFromInt(amount), nil).Times(1)

	suite.stakingKeeper.EXPECT().
		Undelegate(
			gomock.Any(),
			authtypes.NewModuleAddress(moduleName),
			valAddr,
			math.LegacyNewDecFromInt(amount),
		).Return(suite.ctx.BlockTime().Add(time.Hour*24*21), amount, nil).Times(1)

	suite.tierKeeper.SetParams(suite.ctx, params)

	// add a lockup and verify that it exists before trying to unlock
	suite.tierKeeper.AddLockup(suite.ctx, delAddr, valAddr, amount)
	lockedAmt := suite.tierKeeper.GetLockupAmount(suite.ctx, delAddr, valAddr)
	suite.Require().Equal(amount, lockedAmt, "expected lockup amount to be set")

	// perform unlock and verify that unlocking lockup is set correctly
	creationHeight, completionTime, unlockTime, err := suite.tierKeeper.Unlock(suite.ctx, delAddr, valAddr, amount)
	suite.Require().NoError(err)

	suite.Require().Equal(suite.ctx.BlockHeight(), creationHeight)
	suite.Require().Equal(expectedCompletionTime, completionTime)
	suite.Require().Equal(expectedUnlockTime, unlockTime)
}

// TestRedelegate is using mock keepers to verify that required function calls are made as expected on Redelegate().
func (suite *KeeperTestSuite) TestRedelegate() {
	amount := math.NewInt(1000)
	shares := math.LegacyNewDecFromInt(amount)
	expectedCompletionTime := suite.ctx.BlockTime().Add(time.Hour * 24 * 21)

	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	suite.Require().NoError(err)
	srcValAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	suite.Require().NoError(err)
	dstValAddr, err := sdk.ValAddressFromBech32("sourcevaloper13fj7t2yptf9k6ad6fv38434znzay4s4pjk0r4f")
	suite.Require().NoError(err)

	// add initial lockup to the source validator
	suite.tierKeeper.AddLockup(suite.ctx, delAddr, srcValAddr, amount)

	suite.stakingKeeper.EXPECT().
		ValidateUnbondAmount(gomock.Any(), authtypes.NewModuleAddress(types.ModuleName), srcValAddr, amount).
		Return(shares, nil).Times(1)

	suite.stakingKeeper.EXPECT().
		BeginRedelegation(gomock.Any(), authtypes.NewModuleAddress(types.ModuleName), srcValAddr, dstValAddr, shares).
		Return(expectedCompletionTime, nil).Times(1)

	// perform redelegate and verify that lockups were updated successfully
	completionTime, err := suite.tierKeeper.Redelegate(suite.ctx, delAddr, srcValAddr, dstValAddr, amount)
	suite.Require().NoError(err)
	suite.Require().Equal(expectedCompletionTime, completionTime)

	srcLockedAmt := suite.tierKeeper.GetLockupAmount(suite.ctx, delAddr, srcValAddr)
	dstLockedAmt := suite.tierKeeper.GetLockupAmount(suite.ctx, delAddr, dstValAddr)
	suite.Require().Equal(math.ZeroInt(), srcLockedAmt, "source validator lockup should be zero")
	suite.Require().Equal(amount, dstLockedAmt, "destination validator lockup should match the redelegated amount")
}
