package tier

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/sourcenetwork/sourcehub/app"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
	keepertest "github.com/sourcenetwork/sourcehub/testutil/keeper"
	"github.com/sourcenetwork/sourcehub/x/feegrant"
	tierkeeper "github.com/sourcenetwork/sourcehub/x/tier/keeper"
	tiertypes "github.com/sourcenetwork/sourcehub/x/tier/types"
)

type FeegrantIntegrationTestSuite struct {
	suite.Suite

	ctx       sdk.Context
	keeper    tierkeeper.Keeper
	msgServer tiertypes.MsgServer
}

func (suite *FeegrantIntegrationTestSuite) SetupTest() {
	app.SetConfig(false)

	ResetTestAddrIndices()

	k, ctx := keepertest.TierKeeper(suite.T())
	suite.ctx = ctx
	suite.keeper = k
	suite.msgServer = tierkeeper.NewMsgServerImpl(&k)
}

func (suite *FeegrantIntegrationTestSuite) createTestAddresses() (developer, user sdk.AccAddress) {
	return NextPair(suite.T(), &suite.keeper, suite.ctx)
}

func TestFeegrantIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(FeegrantIntegrationTestSuite))
}

func (suite *FeegrantIntegrationTestSuite) TestPeriodicAllowanceCreation() {
	suite.T().Run("Periodic allowance should be created with subscription", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()
		userDid := NextUserDid()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeDelegator(t, &suite.keeper, suite.ctx, developer, math.NewInt(2_000_000))
		keepertest.InitializeValidator(t, suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(10_000)))

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
			Amount:    amount,
			Period:    3600,
		}

		resp, err := suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		sub := suite.keeper.GetUserSubscription(suite.ctx, developer, userDid)
		require.NotNil(t, sub)

		allowance, err := suite.keeper.GetFeegrantKeeper().GetDIDAllowance(suite.ctx, developer, userDid)
		require.NoError(t, err)
		require.NotNil(t, allowance)
		_, ok := allowance.(*feegrant.PeriodicAllowance)
		require.True(t, ok, "Expected PeriodicAllowance type")
	})
}

func (suite *FeegrantIntegrationTestSuite) TestPeriodicAllowanceWithZeroPeriod() {
	suite.T().Run("Zero period should use default epoch duration", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()
		userDid := NextUserDid()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeDelegator(t, &suite.keeper, suite.ctx, developer, math.NewInt(2_000_000))
		keepertest.InitializeValidator(t, suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(10_000)))

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
			Amount:    amount,
			Period:    0,
		}

		resp, err := suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		sub := suite.keeper.GetUserSubscription(suite.ctx, developer, userDid)
		require.NotNil(t, sub)
		require.Equal(t, uint64(0), sub.Period)

		allowance, err := suite.keeper.GetFeegrantKeeper().GetDIDAllowance(suite.ctx, developer, userDid)
		require.NoError(t, err)
		periodicAllowance, ok := allowance.(*feegrant.PeriodicAllowance)
		require.True(t, ok, "Expected PeriodicAllowance type")
		params := suite.keeper.GetParams(suite.ctx)
		expectedPeriod := *params.EpochDuration
		require.Equal(t, expectedPeriod, periodicAllowance.Period)
	})
}

func (suite *FeegrantIntegrationTestSuite) TestAllowanceUpdateOnSubscriptionUpdate() {
	suite.T().Run("Allowance should be updated when subscription is updated", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()
		userDid := NextUserDid()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeDelegator(t, &suite.keeper, suite.ctx, developer, math.NewInt(5_000_000))
		keepertest.InitializeValidator(t, suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(5_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(20_000)))

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
			Amount:    amount,
			Period:    3600,
		}
		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.NoError(t, err)

		newAmount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(2000))
		updateMsg := &tiertypes.MsgUpdateUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
			Amount:    newAmount,
			Period:    7200,
		}

		resp, err := suite.msgServer.UpdateUserSubscription(suite.ctx, updateMsg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		sub := suite.keeper.GetUserSubscription(suite.ctx, developer, userDid)
		require.NotNil(t, sub)
		require.Equal(t, newAmount, sub.CreditAmount)
		require.Equal(t, uint64(7200), sub.Period)

		allowance, err := suite.keeper.GetFeegrantKeeper().GetDIDAllowance(suite.ctx, developer, userDid)
		require.NoError(t, err)
		periodicAllowance, ok := allowance.(*feegrant.PeriodicAllowance)
		require.True(t, ok, "Expected PeriodicAllowance type")
		expectedPeriod := time.Duration(7200) * time.Second
		require.Equal(t, expectedPeriod, periodicAllowance.Period)
		expectedSpendLimit := sdk.NewCoins(newAmount)
		require.Equal(t, expectedSpendLimit, periodicAllowance.Basic.SpendLimit)
	})
}

func (suite *FeegrantIntegrationTestSuite) TestAllowanceRemovalOnSubscriptionRemoval() {
	suite.T().Run("Allowance should be removed when subscription is removed", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()
		userDid := NextUserDid()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeDelegator(t, &suite.keeper, suite.ctx, developer, math.NewInt(2_000_000))
		keepertest.InitializeValidator(t, suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(10_000)))

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
			Amount:    amount,
			Period:    3600,
		}
		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.NoError(t, err)

		allowance, err := suite.keeper.GetFeegrantKeeper().GetDIDAllowance(suite.ctx, developer, userDid)
		require.NoError(t, err)
		require.NotNil(t, allowance)

		removeMsg := &tiertypes.MsgRemoveUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
		}
		_, err = suite.msgServer.RemoveUserSubscription(suite.ctx, removeMsg)
		require.NoError(t, err)

		sub := suite.keeper.GetUserSubscription(suite.ctx, developer, userDid)
		require.Nil(t, sub)
	})
}

func (suite *FeegrantIntegrationTestSuite) TestZeroAmountValidation() {
	suite.T().Run("Zero amount should be rejected by validation", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()
		userDid := NextUserDid()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.ZeroInt())
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
			Amount:    amount,
			Period:    3600,
		}

		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.Error(t, err)
		require.ErrorContains(t, err, "amount must be positive")
	})
}

func (suite *FeegrantIntegrationTestSuite) TestAllowanceCleanupOnDeveloperRemoval() {
	suite.T().Run("All allowances should be cleaned up when developer is removed", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()
		userDid := NextUserDid()
		user2Did := "did:key:bob"

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeDelegator(t, &suite.keeper, suite.ctx, developer, math.NewInt(5_000_000))
		keepertest.InitializeValidator(t, suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(5_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(20_000)))

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))

		addMsg1 := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
			Amount:    amount,
			Period:    3600,
		}
		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg1)
		require.NoError(t, err)

		addMsg2 := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   user2Did,
			Amount:    amount,
			Period:    3600,
		}
		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg2)
		require.NoError(t, err)

		removeMsg := &tiertypes.MsgRemoveDeveloper{
			Developer: developer.String(),
		}
		_, err = suite.msgServer.RemoveDeveloper(suite.ctx, removeMsg)
		require.NoError(t, err)

		sub1 := suite.keeper.GetUserSubscription(suite.ctx, developer, userDid)
		require.Nil(t, sub1)
		sub2 := suite.keeper.GetUserSubscription(suite.ctx, developer, user2Did)
		require.Nil(t, sub2)
	})
}
