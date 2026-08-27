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
	keepertest "github.com/sourcenetwork/vera/testutil/keeper"
	tierkeeper "github.com/sourcenetwork/vera/x/tier/keeper"
	tiertypes "github.com/sourcenetwork/vera/x/tier/types"
)

type EdgeCasesTestSuite struct {
	suite.Suite

	ctx         sdk.Context
	keeper      tierkeeper.Keeper
	msgServer   tiertypes.MsgServer
	addrFactory *TestAddressFactory
}

func (suite *EdgeCasesTestSuite) SetupTest() {
	app.SetConfig(false)

	k, ctx := keepertest.TierKeeper(suite.T())
	suite.ctx = ctx
	suite.keeper = k
	suite.msgServer = tierkeeper.NewMsgServerImpl(&k)
	suite.addrFactory = NewTestAddressFactory()
}

func (suite *EdgeCasesTestSuite) createTestAddresses() (developer, user sdk.AccAddress) {
	return suite.addrFactory.NextPair(suite.T(), &suite.keeper, suite.ctx)
}

func TestEdgeCasesTestSuite(t *testing.T) {
	suite.Run(t, new(EdgeCasesTestSuite))
}

func (s *EdgeCasesTestSuite) TestInvalidDeveloperAddressInCreateDeveloper() {
	msg := &tiertypes.MsgCreateDeveloper{
		Developer:       "vera1invalidaddress123456789012345678901234567890",
		AutoLockEnabled: true,
	}

	err := msg.ValidateBasic()
	require.Error(s.T(), err)
}

func (s *EdgeCasesTestSuite) TestEmptyDeveloperAddress() {
	msg := &tiertypes.MsgCreateDeveloper{
		Developer:       "",
		AutoLockEnabled: true,
	}

	err := msg.ValidateBasic()
	require.Error(s.T(), err)
}

func (s *EdgeCasesTestSuite) TestEmptyUserDid() {
	developer, _ := s.createTestAddresses()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	amount := uint64(1000)
	addMsg := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   "",
		Amount:    amount,
		Period:    3600,
	}

	err = addMsg.ValidateBasic()
	require.Error(s.T(), err)
}

func (s *EdgeCasesTestSuite) TestMaximumAmountSubscription() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	maxBalance := math.NewInt(1000000000000000000)

	valAddr, err := sdk.ValAddressFromBech32("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, maxBalance)
	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, maxBalance)

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	require.NoError(s.T(), s.keeper.Lock(s.ctx, developer, valAddr, maxBalance))

	amount := uint64(1000000000000000000)
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
	require.Equal(s.T(), amount, sub.CreditAmount)
}

func (s *EdgeCasesTestSuite) TestMaximumPeriodValue() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	valAddr, err := sdk.ValAddressFromBech32("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, math.NewInt(2_000_000))
	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, math.NewInt(1_000_000))

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	require.NoError(s.T(), s.keeper.Lock(s.ctx, developer, valAddr, math.NewInt(10_000)))

	amount := uint64(1000)
	maxPeriod := uint64(365 * 24 * 3600)
	addMsg := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount,
		Period:    maxPeriod,
	}

	resp, err := s.msgServer.AddUserSubscription(s.ctx, addMsg)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)

	sub := s.keeper.GetUserSubscription(s.ctx, developer, userDid)
	require.NotNil(s.T(), sub)
	require.Equal(s.T(), maxPeriod, sub.Period)
}

func (s *EdgeCasesTestSuite) TestMultipleSubscriptionsForSameUser() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	valAddr, err := sdk.ValAddressFromBech32("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, math.NewInt(1_000_000))
	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, math.NewInt(1_000_000))

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	require.NoError(s.T(), s.keeper.Lock(s.ctx, developer, valAddr, math.NewInt(10_000)))

	amount1 := uint64(1000)
	addMsg1 := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount1,
		Period:    3600,
	}
	_, err = s.msgServer.AddUserSubscription(s.ctx, addMsg1)
	require.NoError(s.T(), err)

	amount2 := uint64(2000)
	addMsg2 := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount2,
		Period:    7200,
	}
	_, err = s.msgServer.AddUserSubscription(s.ctx, addMsg2)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "already subscribed")
}

func (s *EdgeCasesTestSuite) TestUpdateThenRemoveSubscription() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	valAddr, err := sdk.ValAddressFromBech32("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, math.NewInt(1_000_000))
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

	newAmount := uint64(2000)
	updateMsg := &tiertypes.MsgUpdateUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    newAmount,
		Period:    7200,
	}
	_, err = s.msgServer.UpdateUserSubscription(s.ctx, updateMsg)
	require.NoError(s.T(), err)

	removeMsg := &tiertypes.MsgRemoveUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
	}
	_, err = s.msgServer.RemoveUserSubscription(s.ctx, removeMsg)
	require.NoError(s.T(), err)

	sub := s.keeper.GetUserSubscription(s.ctx, developer, userDid)
	require.Nil(s.T(), sub)
}

func (s *EdgeCasesTestSuite) TestDeveloperRemovalCleansUpAllState() {
	developer := s.addrFactory.NextDeveloper(s.T(), &s.keeper, s.ctx)

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	valAddr, err := sdk.ValAddressFromBech32("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, math.NewInt(2_000_000))
	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, math.NewInt(1_000_000))

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	require.NoError(s.T(), s.keeper.Lock(s.ctx, developer, valAddr, math.NewInt(20_000)))

	amount := uint64(1000)

	users := []sdk.AccAddress{
		s.addrFactory.NextUser(s.T(), &s.keeper, s.ctx),
		s.addrFactory.NextUser(s.T(), &s.keeper, s.ctx),
	}

	userDids := []string{
		s.addrFactory.NextUserDid(),
		s.addrFactory.NextUserDid(),
	}

	for i := range users {
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   userDids[i],
			Amount:    amount,
			Period:    3600,
		}
		_, err = s.msgServer.AddUserSubscription(s.ctx, addMsg)
		require.NoError(s.T(), err)
	}

	for _, did := range userDids {
		sub := s.keeper.GetUserSubscription(s.ctx, developer, did)
		require.NotNil(s.T(), sub)
	}

	removeMsg := &tiertypes.MsgRemoveDeveloper{
		Developer: developer.String(),
	}
	_, err = s.msgServer.RemoveDeveloper(s.ctx, removeMsg)
	require.NoError(s.T(), err)

	dev := s.keeper.GetDeveloper(s.ctx, developer)
	require.Nil(s.T(), dev)

	for _, did := range userDids {
		sub := s.keeper.GetUserSubscription(s.ctx, developer, did)
		require.Nil(s.T(), sub)
	}
}

func (s *EdgeCasesTestSuite) TestSubscriptionAmountsAndTotalsAreConsistent() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	valAddr, err := sdk.ValAddressFromBech32("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
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

	sub := s.keeper.GetUserSubscription(s.ctx, developer, userDid)
	require.NotNil(s.T(), sub)
	require.Equal(s.T(), amount, sub.CreditAmount)

	newAmount := uint64(3000)
	updateMsg := &tiertypes.MsgUpdateUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    newAmount,
		Period:    3600,
	}
	_, err = s.msgServer.UpdateUserSubscription(s.ctx, updateMsg)
	require.NoError(s.T(), err)

	sub = s.keeper.GetUserSubscription(s.ctx, developer, userDid)
	require.NotNil(s.T(), sub)
	require.Equal(s.T(), newAmount, sub.CreditAmount)

	smallerAmount := uint64(3000)
	updateMsg2 := &tiertypes.MsgUpdateUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    smallerAmount,
		Period:    3600,
	}
	_, err = s.msgServer.UpdateUserSubscription(s.ctx, updateMsg2)
	require.NoError(s.T(), err)

	sub = s.keeper.GetUserSubscription(s.ctx, developer, userDid)
	require.NotNil(s.T(), sub)
	require.Equal(s.T(), smallerAmount, sub.CreditAmount)
}

func (s *EdgeCasesTestSuite) TestDeveloperWithManySubscriptions() {
	developer, _ := s.createTestAddresses()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	valAddr, err := sdk.ValAddressFromBech32("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, math.NewInt(2_000_000))
	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, math.NewInt(1_000_000))

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	require.NoError(s.T(), s.keeper.Lock(s.ctx, developer, valAddr, math.NewInt(50_000)))

	numUsers := 2
	users := make([]sdk.AccAddress, numUsers)
	userDids := make([]string, numUsers)
	for i := 0; i < numUsers; i++ {
		userAddr := s.addrFactory.NextUser(s.T(), &s.keeper, s.ctx)
		users[i] = userAddr
		userDids[i] = s.addrFactory.NextUserDid()

		amount := uint64(100)
		addMsg := &tiertypes.MsgAddUserSubscription{
			Developer: developer.String(),
			UserDid:   userDids[i],
			Amount:    amount,
			Period:    3600,
		}
		_, err = s.msgServer.AddUserSubscription(s.ctx, addMsg)
		require.NoError(s.T(), err)
	}

	for _, did := range userDids {
		sub := s.keeper.GetUserSubscription(s.ctx, developer, did)
		require.NotNil(s.T(), sub)
	}

	removeMsg := &tiertypes.MsgRemoveDeveloper{
		Developer: developer.String(),
	}
	_, err = s.msgServer.RemoveDeveloper(s.ctx, removeMsg)
	require.NoError(s.T(), err)

	for _, did := range userDids {
		sub := s.keeper.GetUserSubscription(s.ctx, developer, did)
		require.Nil(s.T(), sub)
	}
}
