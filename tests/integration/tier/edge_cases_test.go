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

type EdgeCasesTestSuite struct {
	suite.Suite

	ctx       sdk.Context
	keeper    tierkeeper.Keeper
	msgServer tiertypes.MsgServer
}

func (suite *EdgeCasesTestSuite) SetupTest() {
	app.SetConfig(false)

	ResetTestAddrIndices()

	k, ctx := keepertest.TierKeeper(suite.T())
	suite.ctx = ctx
	suite.keeper = k
	suite.msgServer = tierkeeper.NewMsgServerImpl(&k)
}

func (suite *EdgeCasesTestSuite) createTestAddresses() (developer, user sdk.AccAddress) {
	return NextPair(suite.T(), &suite.keeper, suite.ctx)
}

func TestEdgeCasesTestSuite(t *testing.T) {
	suite.Run(t, new(EdgeCasesTestSuite))
}

func (suite *EdgeCasesTestSuite) TestInvalidAddresses() {
	suite.T().Run("Invalid developer address in CreateDeveloper", func(t *testing.T) {
		msg := &tiertypes.MsgCreateDeveloper{
			Developer:       "source1invalidaddress123456789012345678901234567890",
			AutoLockEnabled: true,
		}

		err := msg.ValidateBasic()
		require.Error(t, err)
	})

	suite.T().Run("Empty developer address", func(t *testing.T) {
		msg := &tiertypes.MsgCreateDeveloper{
			Developer:       "",
			AutoLockEnabled: true,
		}

		err := msg.ValidateBasic()
		require.Error(t, err)
	})

	suite.T().Run("Invalid user address in AddUserSubscription", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			User:      "source1invaliduser123456789012345678901234567890",
			Amount:    amount,
			Period:    3600,
		}

		err = addMsg.ValidateBasic()
		require.Error(t, err)
	})

	suite.T().Run("Empty user address", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			User:      "",
			Amount:    amount,
			Period:    3600,
		}

		err = addMsg.ValidateBasic()
		require.Error(t, err)
	})
}

func (suite *EdgeCasesTestSuite) TestBoundaryValues() {
	suite.T().Run("Maximum amount subscription", func(t *testing.T) {
		developer, user := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		maxBalance := math.NewInt(1000000000000000000)

		valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeDelegator(suite.T(), &suite.keeper, suite.ctx, developer, maxBalance)
		keepertest.InitializeValidator(suite.T(), suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, maxBalance)

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, maxBalance))

		amount := sdk.NewCoin(appparams.MicroCreditDenom, maxBalance)
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

	suite.T().Run("Maximum period value", func(t *testing.T) {
		developer, user := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeDelegator(suite.T(), &suite.keeper, suite.ctx, developer, math.NewInt(2_000_000))
		keepertest.InitializeValidator(suite.T(), suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(10_000)))

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		maxPeriod := uint64(365 * 24 * 3600)
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			User:      user.String(),
			Amount:    amount,
			Period:    maxPeriod,
		}

		resp, err := suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		sub := suite.keeper.GetUserSubscription(suite.ctx, developer, user)
		require.NotNil(t, sub)
		require.Equal(t, maxPeriod, sub.Period)
	})

	suite.T().Run("Negative amount should be prevented by validation", func(t *testing.T) {
		developer, user := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		amount := sdk.Coin{Denom: appparams.MicroCreditDenom, Amount: math.NewInt(-1000)}
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			User:      user.String(),
			Amount:    amount,
			Period:    3600,
		}

		err = addMsg.ValidateBasic()
		require.Error(t, err)
	})
}

func (suite *EdgeCasesTestSuite) TestConcurrentOperations() {
	suite.T().Run("Multiple subscriptions for same user", func(t *testing.T) {
		developer, user := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeDelegator(suite.T(), &suite.keeper, suite.ctx, developer, math.NewInt(1_000_000))
		keepertest.InitializeValidator(suite.T(), suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(10_000)))

		amount1 := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg1 := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			User:      user.String(),
			Amount:    amount1,
			Period:    3600,
		}
		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg1)
		require.NoError(t, err)

		amount2 := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(2000))
		addMsg2 := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			User:      user.String(),
			Amount:    amount2,
			Period:    7200,
		}
		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg2)
		require.Error(t, err)
		require.ErrorContains(t, err, "already subscribed")
	})

	suite.T().Run("Update then remove subscription", func(t *testing.T) {
		developer, user := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeDelegator(suite.T(), &suite.keeper, suite.ctx, developer, math.NewInt(1_000_000))
		keepertest.InitializeValidator(suite.T(), suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(10_000)))

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			User:      user.String(),
			Amount:    amount,
			Period:    3600,
		}
		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.NoError(t, err)

		newAmount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(2000))
		updateMsg := &tiertypes.MsgUpdateUserSubscription{
			Developer: developer.String(),
			User:      user.String(),
			Amount:    newAmount,
			Period:    7200,
		}
		_, err = suite.msgServer.UpdateUserSubscription(suite.ctx, updateMsg)
		require.NoError(t, err)

		removeMsg := &tiertypes.MsgRemoveUserSubscription{
			Developer: developer.String(),
			User:      user.String(),
		}
		_, err = suite.msgServer.RemoveUserSubscription(suite.ctx, removeMsg)
		require.NoError(t, err)

		sub := suite.keeper.GetUserSubscription(suite.ctx, developer, user)
		require.Nil(t, sub)
	})
}

func (suite *EdgeCasesTestSuite) TestStateConsistency() {
	suite.T().Run("Developer removal cleans up all state", func(t *testing.T) {
		developer := NextDeveloper(suite.T(), &suite.keeper, suite.ctx)

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeDelegator(suite.T(), &suite.keeper, suite.ctx, developer, math.NewInt(2_000_000))
		keepertest.InitializeValidator(suite.T(), suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(20_000)))

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))

		users := []sdk.AccAddress{
			NextUser(suite.T(), &suite.keeper, suite.ctx),
			NextUser(suite.T(), &suite.keeper, suite.ctx),
		}

		for _, u := range users {
			addMsg := &tiertypes.MsgAddUserSubscription{
				Developer: developer.String(),
				User:      u.String(),
				Amount:    amount,
				Period:    3600,
			}
			_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
			require.NoError(t, err)
		}

		for _, u := range users {
			sub := suite.keeper.GetUserSubscription(suite.ctx, developer, u)
			require.NotNil(t, sub)
		}

		removeMsg := &tiertypes.MsgRemoveDeveloper{
			Developer: developer.String(),
		}
		_, err = suite.msgServer.RemoveDeveloper(suite.ctx, removeMsg)
		require.NoError(t, err)

		dev := suite.keeper.GetDeveloper(suite.ctx, developer)
		require.Nil(t, dev)

		for _, u := range users {
			sub := suite.keeper.GetUserSubscription(suite.ctx, developer, u)
			require.Nil(t, sub)
		}
	})

	suite.T().Run("Subscription amounts and totals are consistent", func(t *testing.T) {
		developer, user := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeDelegator(suite.T(), &suite.keeper, suite.ctx, developer, math.NewInt(2_000_000))
		keepertest.InitializeValidator(suite.T(), suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(10_000)))

		subscriptionAmount := math.NewInt(2000)
		amount := sdk.NewCoin(appparams.MicroCreditDenom, subscriptionAmount)
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			User:      user.String(),
			Amount:    amount,
			Period:    3600,
		}
		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.NoError(t, err)

		sub := suite.keeper.GetUserSubscription(suite.ctx, developer, user)
		require.NotNil(t, sub)
		require.Equal(t, subscriptionAmount, sub.CreditAmount.Amount)

		newSubscriptionAmount := math.NewInt(3000)
		newAmount := sdk.NewCoin(appparams.MicroCreditDenom, newSubscriptionAmount)
		updateMsg := &tiertypes.MsgUpdateUserSubscription{
			Developer: developer.String(),
			User:      user.String(),
			Amount:    newAmount,
			Period:    3600,
		}
		_, err = suite.msgServer.UpdateUserSubscription(suite.ctx, updateMsg)
		require.NoError(t, err)

		sub = suite.keeper.GetUserSubscription(suite.ctx, developer, user)
		require.NotNil(t, sub)
		require.Equal(t, newSubscriptionAmount, sub.CreditAmount.Amount)

		smallerAmount := math.NewInt(1000)
		smallerCoin := sdk.NewCoin(appparams.MicroCreditDenom, smallerAmount)
		updateMsg2 := &tiertypes.MsgUpdateUserSubscription{
			Developer: developer.String(),
			User:      user.String(),
			Amount:    smallerCoin,
			Period:    3600,
		}
		_, err = suite.msgServer.UpdateUserSubscription(suite.ctx, updateMsg2)
		require.NoError(t, err)

		sub = suite.keeper.GetUserSubscription(suite.ctx, developer, user)
		require.NotNil(t, sub)
		require.Equal(t, smallerAmount, sub.CreditAmount.Amount)
	})
}

func (suite *EdgeCasesTestSuite) TestResourceLimits() {
	suite.T().Run("Developer with many subscriptions", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
		require.NoError(t, err)

		keepertest.InitializeDelegator(suite.T(), &suite.keeper, suite.ctx, developer, math.NewInt(2_000_000))
		keepertest.InitializeValidator(suite.T(), suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(50_000)))

		numUsers := 2
		users := make([]sdk.AccAddress, numUsers)
		for i := 0; i < numUsers; i++ {
			userAddr := NextUser(suite.T(), &suite.keeper, suite.ctx)
			users[i] = userAddr

			amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(100))
			addMsg := &tiertypes.MsgAddUserSubscription{
				Developer: developer.String(),
				User:      userAddr.String(),
				Amount:    amount,
				Period:    3600,
			}
			_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
			require.NoError(t, err)
		}

		for _, userAddr := range users {
			sub := suite.keeper.GetUserSubscription(suite.ctx, developer, userAddr)
			require.NotNil(t, sub)
		}

		removeMsg := &tiertypes.MsgRemoveDeveloper{
			Developer: developer.String(),
		}
		_, err = suite.msgServer.RemoveDeveloper(suite.ctx, removeMsg)
		require.NoError(t, err)

		for _, userAddr := range users {
			sub := suite.keeper.GetUserSubscription(suite.ctx, developer, userAddr)
			require.Nil(t, sub)
		}
	})
}
