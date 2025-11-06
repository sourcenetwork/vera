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

type TierIntegrationTestSuite struct {
	suite.Suite

	ctx       sdk.Context
	keeper    tierkeeper.Keeper
	msgServer tiertypes.MsgServer
}

func (suite *TierIntegrationTestSuite) SetupTest() {
	app.SetConfig(false)

	ResetTestAddrIndices()

	k, ctx := keepertest.TierKeeper(suite.T())
	suite.ctx = ctx
	suite.keeper = k
	suite.msgServer = tierkeeper.NewMsgServerImpl(&k)
}

func (suite *TierIntegrationTestSuite) createTestAddresses() (developer, user sdk.AccAddress) {
	return NextPair(suite.T(), &suite.keeper, suite.ctx)
}

func TestTierIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(TierIntegrationTestSuite))
}

func (suite *TierIntegrationTestSuite) TestCreateDeveloper() {
	suite.T().Run("Valid developer creation", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()

		msg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: true,
		}

		resp, err := suite.msgServer.CreateDeveloper(suite.ctx, msg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		dev := suite.keeper.GetDeveloper(suite.ctx, developer)
		require.NotNil(t, dev)
		require.Equal(t, developer.String(), dev.Address)
		require.True(t, dev.AutoLockEnabled)
	})

	suite.T().Run("Cannot create duplicate developer", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()

		msg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}

		_, err := suite.msgServer.CreateDeveloper(suite.ctx, msg)
		require.NoError(t, err)

		_, err = suite.msgServer.CreateDeveloper(suite.ctx, msg)
		require.Error(t, err)
		require.ErrorContains(t, err, "already exists")
	})

	suite.T().Run("Invalid developer address", func(t *testing.T) {
		msg := &tiertypes.MsgCreateDeveloper{
			Developer:       "cosmos1invalidaddress123456789012345678901234567890",
			AutoLockEnabled: true,
		}

		err := msg.ValidateBasic()
		require.Error(t, err)
	})
}

func (suite *TierIntegrationTestSuite) TestUpdateDeveloper() {
	suite.T().Run("Update existing developer", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		updateMsg := &tiertypes.MsgUpdateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: true,
		}
		resp, err := suite.msgServer.UpdateDeveloper(suite.ctx, updateMsg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		dev := suite.keeper.GetDeveloper(suite.ctx, developer)
		require.NotNil(t, dev)
		require.True(t, dev.AutoLockEnabled)
	})

	suite.T().Run("Update non-existent developer fails", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()

		updateMsg := &tiertypes.MsgUpdateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: true,
		}
		_, err := suite.msgServer.UpdateDeveloper(suite.ctx, updateMsg)
		require.Error(t, err)
		require.ErrorContains(t, err, "does not exist")
	})
}

func (suite *TierIntegrationTestSuite) TestRemoveDeveloper() {
	suite.T().Run("Remove existing developer", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: true,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		removeMsg := &tiertypes.MsgRemoveDeveloper{
			Developer: developer.String(),
		}
		resp, err := suite.msgServer.RemoveDeveloper(suite.ctx, removeMsg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		dev := suite.keeper.GetDeveloper(suite.ctx, developer)
		require.Nil(t, dev)
	})

	suite.T().Run("Remove non-existent developer", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()

		removeMsg := &tiertypes.MsgRemoveDeveloper{
			Developer: developer.String(),
		}
		_, err := suite.msgServer.RemoveDeveloper(suite.ctx, removeMsg)
		require.Error(t, err)
		require.ErrorContains(t, err, "does not exist")
	})
}

func (suite *TierIntegrationTestSuite) TestAddUserSubscription() {
	suite.T().Run("Valid user subscription", func(t *testing.T) {
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

		keepertest.InitializeDelegator(t, &suite.keeper, suite.ctx, developer, math.NewInt(1000000))
		keepertest.InitializeValidator(t, suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(1000)))

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
		require.Equal(t, developer.String(), sub.Developer)
		require.Equal(t, userDid, sub.UserDid)
		require.Equal(t, amount, sub.CreditAmount)
		require.Equal(t, uint64(3600), sub.Period)
	})

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

	suite.T().Run("Cannot add duplicate subscription", func(t *testing.T) {
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

		keepertest.InitializeDelegator(t, &suite.keeper, suite.ctx, developer, math.NewInt(1_000_000))
		keepertest.InitializeValidator(t, suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(1000)))

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
			Amount:    amount,
			Period:    3600,
		}

		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.NoError(t, err)

		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.Error(t, err)
		require.ErrorContains(t, err, "already subscribed")
	})

	suite.T().Run("Non-existent developer", func(t *testing.T) {
		developer := NextDeveloper(suite.T(), &suite.keeper, suite.ctx)
		userDid := NextUserDid()

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
			Amount:    amount,
			Period:    3600,
		}

		_, err := suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.Error(t, err)
		require.ErrorContains(t, err, "not found")
	})

	suite.T().Run("Invalid credit denomination", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()
		userDid := NextUserDid()

		createMsg := &tiertypes.MsgCreateDeveloper{
			Developer:       developer.String(),
			AutoLockEnabled: false,
		}
		_, err := suite.msgServer.CreateDeveloper(suite.ctx, createMsg)
		require.NoError(t, err)

		amount := sdk.NewCoin("invalid_denom", math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
			Amount:    amount,
			Period:    3600,
		}

		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.Error(t, err)
		require.ErrorContains(t, err, "invalid amount denomination")
	})
}

func (suite *TierIntegrationTestSuite) TestUpdateUserSubscription() {
	suite.T().Run("Update existing subscription", func(t *testing.T) {
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

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(2000)))

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
	})

	suite.T().Run("Update non-existent subscription", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()
		userDid := NextUserDid()

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		updateMsg := &tiertypes.MsgUpdateUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
			Amount:    amount,
			Period:    3600,
		}

		_, err := suite.msgServer.UpdateUserSubscription(suite.ctx, updateMsg)
		require.Error(t, err)
		require.ErrorContains(t, err, "not subscribed")
	})
}

func (suite *TierIntegrationTestSuite) TestRemoveUserSubscription() {
	suite.T().Run("Remove existing subscription", func(t *testing.T) {
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

		keepertest.InitializeDelegator(t, &suite.keeper, suite.ctx, developer, math.NewInt(1_000_000))
		keepertest.InitializeValidator(t, suite.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), suite.ctx, valAddr, math.NewInt(1_000_000))

		suite.ctx = suite.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(1000)))

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
			Amount:    amount,
			Period:    3600,
		}
		_, err = suite.msgServer.AddUserSubscription(suite.ctx, addMsg)
		require.NoError(t, err)

		removeMsg := &tiertypes.MsgRemoveUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
		}

		resp, err := suite.msgServer.RemoveUserSubscription(suite.ctx, removeMsg)
		require.NoError(t, err)
		require.NotNil(t, resp)

		sub := suite.keeper.GetUserSubscription(suite.ctx, developer, userDid)
		require.Nil(t, sub)
	})

	suite.T().Run("Remove non-existent subscription", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()
		userDid := NextUserDid()

		removeMsg := &tiertypes.MsgRemoveUserSubscription{
			Developer: developer.String(),
			UserDid:   userDid,
		}

		_, err := suite.msgServer.RemoveUserSubscription(suite.ctx, removeMsg)
		require.Error(t, err)
		require.ErrorContains(t, err, "not subscribed")
	})
}

func (suite *TierIntegrationTestSuite) TestRemoveDeveloperWithSubscriptions() {
	suite.T().Run("Remove developer should remove all subscriptions", func(t *testing.T) {
		developer, _ := suite.createTestAddresses()
		user1Did := "did:key:alice"
		user2Did := "did:key:bob"

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

		require.NoError(t, suite.keeper.Lock(suite.ctx, developer, valAddr, math.NewInt(2000)))

		amount := sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(1000))
		addMsg1 := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   user1Did,
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

		sub1 := suite.keeper.GetUserSubscription(suite.ctx, developer, user1Did)
		require.NotNil(t, sub1)
		sub2 := suite.keeper.GetUserSubscription(suite.ctx, developer, user2Did)
		require.NotNil(t, sub2)

		removeMsg := &tiertypes.MsgRemoveDeveloper{
			Developer: developer.String(),
		}
		_, err = suite.msgServer.RemoveDeveloper(suite.ctx, removeMsg)
		require.NoError(t, err)

		dev := suite.keeper.GetDeveloper(suite.ctx, developer)
		require.Nil(t, dev)
		sub1 = suite.keeper.GetUserSubscription(suite.ctx, developer, user1Did)
		require.Nil(t, sub1)
		sub2 = suite.keeper.GetUserSubscription(suite.ctx, developer, user2Did)
		require.Nil(t, sub2)
	})
}
