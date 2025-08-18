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
	tierkeeper "github.com/sourcenetwork/sourcehub/x/tier/keeper"
	tiertypes "github.com/sourcenetwork/sourcehub/x/tier/types"
)

type AutoLockIntegrationTestSuite struct {
	suite.Suite

	ctx       sdk.Context
	keeper    tierkeeper.Keeper
	msgServer tiertypes.MsgServer
}

func (suite *AutoLockIntegrationTestSuite) SetupTest() {
	app.SetConfig(false)

	ResetTestAddrIndices()

	k, ctx := keepertest.TierKeeper(suite.T())
	suite.ctx = ctx
	suite.keeper = k
	suite.msgServer = tierkeeper.NewMsgServerImpl(&k)
}

func (suite *AutoLockIntegrationTestSuite) createTestAddresses() (developer, user, validator sdk.AccAddress) {
	developer, user = NextPair(suite.T(), &suite.keeper, suite.ctx)
	validator = sdk.AccAddress(TestValidatorAddr)
	keepertest.CreateAccount(suite.T(), &suite.keeper, suite.ctx, validator)
	return developer, user, validator
}

func TestAutoLockIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AutoLockIntegrationTestSuite))
}

func (suite *AutoLockIntegrationTestSuite) TestAutoLockDisabled() {
	suite.T().Run("Auto-lock disabled - insufficient credits should fail", func(t *testing.T) {
		developer, user, _ := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		keepertest.InitializeDelegator(t, &suite.keeper, suite.ctx, developer, math.NewInt(500))

		valAddr := sdk.ValAddress("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeValidator(t, suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1000000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			User:      user.String(),
			Amount:    amount,
			Period:    3600,
		}

		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.Error(t, err)
		require.ErrorContains(t, err, "insufficient credits and auto-lock disabled")
	})
}

func (suite *AutoLockIntegrationTestSuite) TestAutoLockEnabled() {
	suite.T().Run("Auto-lock enabled - should succeed with lockup", func(t *testing.T) {
		developer, user, _ := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: true,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		keepertest.InitializeDelegator(t, &suite.keeper, suite.ctx, developer, math.NewInt(10000000))

		valAddr := sdk.ValAddress("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		keepertest.InitializeValidator(t, suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1000000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			User:      user.String(),
			Amount:    amount,
			Period:    3600,
		}

		resp, err := suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		sub := suite.keeper.GetUserSubscription(suite.ctx, developer, user)
		require.NotNil(t, sub)
		require.Equal(t, amount, sub.CreditAmount)
	})
}

func (suite *AutoLockIntegrationTestSuite) TestAutoLockEnabledWithSufficientCredits() {
	suite.T().Run("Auto-lock enabled but sufficient credits - should not trigger auto-lock", func(t *testing.T) {
		developer, user, _ := suite.createTestAddresses()

		keepertest.InitializeDelegator(t, &suite.keeper, suite.ctx, developer, math.NewInt(2000))

		valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeValidator(t, suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: true,
		}

		_, err = suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			User:      user.String(),
			Amount:    amount,
			Period:    3600,
		}

		resp, err := suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		sub := suite.keeper.GetUserSubscription(suite.ctx, developer, user)
		require.NotNil(t, sub)
		require.Equal(t, amount, sub.CreditAmount)
	})
}

func (suite *AutoLockIntegrationTestSuite) TestAutoLockToggling() {
	suite.T().Run("Toggle auto-lock setting", func(t *testing.T) {
		developer, _, _ := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		dev := suite.keeper.GetDeveloper(suite.ctx, developer)
		require.NotNil(t, dev)
		require.False(t, dev.AutoLockEnabled)

		updateMsg := &tiertypes.MsgUpdateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: true,
		}
		_, err = suite.msgServer.UpdateDeveloper(suite.ctx, updateMsg)
		require.NoError(t, err)

		dev = suite.keeper.GetDeveloper(suite.ctx, developer)
		require.NotNil(t, dev)
		require.True(t, dev.AutoLockEnabled)

		updateMsg.AutoLockEnabled = false
		_, err = suite.msgServer.UpdateDeveloper(suite.ctx, updateMsg)
		require.NoError(t, err)

		dev = suite.keeper.GetDeveloper(suite.ctx, developer)
		require.NotNil(t, dev)
		require.False(t, dev.AutoLockEnabled)
	})
}
