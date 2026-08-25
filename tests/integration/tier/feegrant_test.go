package tier

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/sourcenetwork/vera/app"
	appparams "github.com/sourcenetwork/vera/app/params"
	keepertest "github.com/sourcenetwork/vera/testutil/keeper"
	"github.com/sourcenetwork/vera/x/feegrant"
	tierkeeper "github.com/sourcenetwork/vera/x/tier/keeper"
	tiertypes "github.com/sourcenetwork/vera/x/tier/types"
)

type FeegrantIntegrationTestSuite struct {
	suite.Suite

	ctx         sdk.Context
	keeper      tierkeeper.Keeper
	msgServer   tiertypes.MsgServer
	addrFactory *TestAddressFactory
}

func (suite *FeegrantIntegrationTestSuite) SetupTest() {
	app.SetConfig(false)

	k, ctx := keepertest.TierKeeper(suite.T())
	suite.ctx = ctx
	suite.keeper = k
	suite.msgServer = tierkeeper.NewMsgServerImpl(&k)
	suite.addrFactory = NewTestAddressFactory()
}

func (suite *FeegrantIntegrationTestSuite) createTestAddresses() (developer, user sdk.AccAddress) {
	return suite.addrFactory.NextPair(suite.T(), &suite.keeper, suite.ctx)
}

func TestFeegrantIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(FeegrantIntegrationTestSuite))
}

func (s *FeegrantIntegrationTestSuite) TestPeriodicAllowanceCreation() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, math.NewInt(2_000_000))
	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, math.NewInt(1_000_000))

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	require.NoError(s.T(), s.keeper.Lock(s.ctx, developer, valAddr, math.NewInt(10_000)))

	amount := uint64(1000)
	addMsg := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount,
		Period:    3600,
	}

	resp, err := s.msgServer.AddUserSubscription(s.ctx, addMsg)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)

	sub := s.keeper.GetUserSubscription(s.ctx, developer, userDid)
	require.NotNil(s.T(), sub)

	allowance, err := s.keeper.GetFeegrantKeeper().GetDIDAllowance(s.ctx, developer, userDid)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), allowance)
	_, ok := allowance.(*feegrant.PeriodicAllowance)
	require.True(s.T(), ok, "Expected PeriodicAllowance type")
}

func (s *FeegrantIntegrationTestSuite) TestPeriodicAllowanceWithZeroPeriod() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, math.NewInt(2_000_000))
	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, math.NewInt(1_000_000))

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	require.NoError(s.T(), s.keeper.Lock(s.ctx, developer, valAddr, math.NewInt(10_000)))

	amount := uint64(1000)
	addMsg := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount,
		Period:    0,
	}

	resp, err := s.msgServer.AddUserSubscription(s.ctx, addMsg)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)

	sub := s.keeper.GetUserSubscription(s.ctx, developer, userDid)
	require.NotNil(s.T(), sub)
	require.Equal(s.T(), uint64(0), sub.Period)

	allowance, err := s.keeper.GetFeegrantKeeper().GetDIDAllowance(s.ctx, developer, userDid)
	require.NoError(s.T(), err)
	periodicAllowance, ok := allowance.(*feegrant.PeriodicAllowance)
	require.True(s.T(), ok, "Expected PeriodicAllowance type")
	params := s.keeper.GetParams(s.ctx)
	expectedPeriod := *params.EpochDuration
	require.Equal(s.T(), expectedPeriod, periodicAllowance.Period)
}

func (s *FeegrantIntegrationTestSuite) TestAllowanceUpdateOnSubscriptionUpdate() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, math.NewInt(5_000_000))
	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, math.NewInt(5_000_000))

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	require.NoError(s.T(), s.keeper.Lock(s.ctx, developer, valAddr, math.NewInt(20_000)))

	amount := uint64(1000)
	addMsg := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount,
		Period:    3600,
	}
	_, err = s.msgServer.AddUserSubscription(s.ctx, addMsg)
	require.NoError(s.T(), err)

	newAmount := uint64(2000)
	updateMsg := &tiertypes.MsgUpdateUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    newAmount,
		Period:    7200,
	}

	resp, err := s.msgServer.UpdateUserSubscription(s.ctx, updateMsg)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)

	sub := s.keeper.GetUserSubscription(s.ctx, developer, userDid)
	require.NotNil(s.T(), sub)
	require.Equal(s.T(), newAmount, sub.CreditAmount)
	require.Equal(s.T(), uint64(7200), sub.Period)

	allowance, err := s.keeper.GetFeegrantKeeper().GetDIDAllowance(s.ctx, developer, userDid)
	require.NoError(s.T(), err)
	periodicAllowance, ok := allowance.(*feegrant.PeriodicAllowance)
	require.True(s.T(), ok, "Expected PeriodicAllowance type")
	expectedPeriod := time.Duration(7200) * time.Second
	require.Equal(s.T(), expectedPeriod, periodicAllowance.Period)
	require.Equal(s.T(), math.NewIntFromUint64(newAmount), periodicAllowance.Basic.SpendLimit.AmountOf(appparams.MicroCreditDenom))
}

func (s *FeegrantIntegrationTestSuite) TestAllowanceRemovalOnSubscriptionRemoval() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, math.NewInt(2_000_000))
	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, math.NewInt(1_000_000))

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	require.NoError(s.T(), s.keeper.Lock(s.ctx, developer, valAddr, math.NewInt(10_000)))

	amount := uint64(1000)
	addMsg := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount,
		Period:    3600,
	}
	_, err = s.msgServer.AddUserSubscription(s.ctx, addMsg)
	require.NoError(s.T(), err)

	allowance, err := s.keeper.GetFeegrantKeeper().GetDIDAllowance(s.ctx, developer, userDid)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), allowance)

	removeMsg := &tiertypes.MsgRemoveUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
	}
	_, err = s.msgServer.RemoveUserSubscription(s.ctx, removeMsg)
	require.NoError(s.T(), err)

	sub := s.keeper.GetUserSubscription(s.ctx, developer, userDid)
	require.Nil(s.T(), sub)
}

func (s *FeegrantIntegrationTestSuite) TestZeroAmountValidation() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	amount := uint64(0)
	addMsg := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount,
		Period:    3600,
	}

	err = addMsg.ValidateBasic()
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "invalid amount")
}

func (s *FeegrantIntegrationTestSuite) TestAllowanceCleanupOnDeveloperRemoval() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()
	user2Did := "did:key:bob"

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, math.NewInt(5_000_000))
	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, math.NewInt(5_000_000))

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	require.NoError(s.T(), s.keeper.Lock(s.ctx, developer, valAddr, math.NewInt(20_000)))

	amount := uint64(1000)

	addMsg1 := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount,
		Period:    3600,
	}
	_, err = s.msgServer.AddUserSubscription(s.ctx, addMsg1)
	require.NoError(s.T(), err)

	addMsg2 := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   user2Did,
		Amount:    amount,
		Period:    3600,
	}
	_, err = s.msgServer.AddUserSubscription(s.ctx, addMsg2)
	require.NoError(s.T(), err)

	removeMsg := &tiertypes.MsgRemoveDeveloper{
		Developer: developer.String(),
	}
	_, err = s.msgServer.RemoveDeveloper(s.ctx, removeMsg)
	require.NoError(s.T(), err)

	sub1 := s.keeper.GetUserSubscription(s.ctx, developer, userDid)
	require.Nil(s.T(), sub1)
	sub2 := s.keeper.GetUserSubscription(s.ctx, developer, user2Did)
	require.Nil(s.T(), sub2)
}
