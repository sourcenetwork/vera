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

type BaseTierTestSuite struct {
	suite.Suite

	ctx         sdk.Context
	keeper      tierkeeper.Keeper
	msgServer   tiertypes.MsgServer
	addrFactory *TestAddressFactory
}

func (s *BaseTierTestSuite) SetupTest() {
	app.SetConfig(false)

	k, ctx := keepertest.TierKeeper(s.T())
	s.ctx = ctx
	s.keeper = k
	s.msgServer = tierkeeper.NewMsgServerImpl(&k)
	s.addrFactory = NewTestAddressFactory()
}

func (s *BaseTierTestSuite) createTestAddresses() (developer, user sdk.AccAddress) {
	return s.addrFactory.NextPair(s.T(), &s.keeper, s.ctx)
}

func (s *BaseTierTestSuite) setupDeveloperWithLockup(developer sdk.AccAddress, lockAmount math.Int) sdk.ValAddress {
	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	valAddr, err := sdk.ValAddressFromBech32("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, lockAmount.MulRaw(1000))
	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, math.NewInt(1_000_000))

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	require.NoError(s.T(), s.keeper.Lock(s.ctx, developer, valAddr, lockAmount))

	return valAddr
}

type DeveloperTestSuite struct {
	BaseTierTestSuite
}

func TestDeveloperTestSuite(t *testing.T) {
	suite.Run(t, new(DeveloperTestSuite))
}

func (s *DeveloperTestSuite) TestCreate() {
	developer, _ := s.createTestAddresses()

	msg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: true,
	}

	resp, err := s.msgServer.CreateDeveloper(s.ctx, msg)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)

	dev := s.keeper.GetDeveloper(s.ctx, developer)
	require.NotNil(s.T(), dev)
	require.Equal(s.T(), developer.String(), dev.Address)
	require.True(s.T(), dev.AutoLockEnabled)
}

func (s *DeveloperTestSuite) TestCreateDuplicate() {
	developer, _ := s.createTestAddresses()

	msg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}

	_, err := s.msgServer.CreateDeveloper(s.ctx, msg)
	require.NoError(s.T(), err)

	_, err = s.msgServer.CreateDeveloper(s.ctx, msg)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "already exists")
}

func (s *DeveloperTestSuite) TestCreateInvalidAddress() {
	msg := &tiertypes.MsgCreateDeveloper{
		Developer:       "cosmos1invalidaddress123456789012345678901234567890",
		AutoLockEnabled: true,
	}

	err := msg.ValidateBasic()
	require.Error(s.T(), err)
}

func (s *DeveloperTestSuite) TestUpdate() {
	developer, _ := s.createTestAddresses()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	updateMsg := &tiertypes.MsgUpdateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: true,
	}
	resp, err := s.msgServer.UpdateDeveloper(s.ctx, updateMsg)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)

	dev := s.keeper.GetDeveloper(s.ctx, developer)
	require.NotNil(s.T(), dev)
	require.True(s.T(), dev.AutoLockEnabled)
}

func (s *DeveloperTestSuite) TestUpdateNonExistent() {
	developer, _ := s.createTestAddresses()

	updateMsg := &tiertypes.MsgUpdateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: true,
	}
	_, err := s.msgServer.UpdateDeveloper(s.ctx, updateMsg)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "does not exist")
}

func (s *DeveloperTestSuite) TestRemove() {
	developer, _ := s.createTestAddresses()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: true,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	removeMsg := &tiertypes.MsgRemoveDeveloper{
		Developer: developer.String(),
	}
	resp, err := s.msgServer.RemoveDeveloper(s.ctx, removeMsg)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)

	dev := s.keeper.GetDeveloper(s.ctx, developer)
	require.Nil(s.T(), dev)
}

func (s *DeveloperTestSuite) TestRemoveNonExistent() {
	developer, _ := s.createTestAddresses()

	removeMsg := &tiertypes.MsgRemoveDeveloper{
		Developer: developer.String(),
	}
	_, err := s.msgServer.RemoveDeveloper(s.ctx, removeMsg)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "does not exist")
}

func (s *DeveloperTestSuite) TestRemoveWithSubscriptions() {
	developer, _ := s.createTestAddresses()
	user1Did := "did:key:alice"
	user2Did := "did:key:bob"

	s.setupDeveloperWithLockup(developer, math.NewInt(2000))

	amount := uint64(1000)
	addMsg1 := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   user1Did,
		Amount:    amount,
		Period:    3600,
	}
	_, err := s.msgServer.AddUserSubscription(s.ctx, addMsg1)
	require.NoError(s.T(), err)

	addMsg2 := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   user2Did,
		Amount:    amount,
		Period:    3600,
	}
	_, err = s.msgServer.AddUserSubscription(s.ctx, addMsg2)
	require.NoError(s.T(), err)

	sub1 := s.keeper.GetUserSubscription(s.ctx, developer, user1Did)
	require.NotNil(s.T(), sub1)
	sub2 := s.keeper.GetUserSubscription(s.ctx, developer, user2Did)
	require.NotNil(s.T(), sub2)

	removeMsg := &tiertypes.MsgRemoveDeveloper{
		Developer: developer.String(),
	}
	_, err = s.msgServer.RemoveDeveloper(s.ctx, removeMsg)
	require.NoError(s.T(), err)

	dev := s.keeper.GetDeveloper(s.ctx, developer)
	require.Nil(s.T(), dev)
	sub1 = s.keeper.GetUserSubscription(s.ctx, developer, user1Did)
	require.Nil(s.T(), sub1)
	sub2 = s.keeper.GetUserSubscription(s.ctx, developer, user2Did)
	require.Nil(s.T(), sub2)
}

// UserSubscriptionTestSuite tests user subscription operations.
type UserSubscriptionTestSuite struct {
	BaseTierTestSuite
}

func TestUserSubscriptionTestSuite(t *testing.T) {
	suite.Run(t, new(UserSubscriptionTestSuite))
}

func (s *UserSubscriptionTestSuite) TestAdd() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	s.setupDeveloperWithLockup(developer, math.NewInt(1000))

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
	require.Equal(s.T(), developer.String(), sub.Developer)
	require.Equal(s.T(), userDid, sub.UserDid)
	require.Equal(s.T(), amount, sub.CreditAmount)
	require.Equal(s.T(), uint64(3600), sub.Period)
}

func (s *UserSubscriptionTestSuite) TestAddZeroAmount() {
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

func (s *UserSubscriptionTestSuite) TestAddDuplicate() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	s.setupDeveloperWithLockup(developer, math.NewInt(1000))

	amount := uint64(1000)
	addMsg := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount,
		Period:    3600,
	}

	_, err := s.msgServer.AddUserSubscription(s.ctx, addMsg)
	require.NoError(s.T(), err)

	_, err = s.msgServer.AddUserSubscription(s.ctx, addMsg)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "already subscribed")
}

func (s *UserSubscriptionTestSuite) TestAddNonExistentDeveloper() {
	developer := s.addrFactory.NextDeveloper(s.T(), &s.keeper, s.ctx)
	userDid := s.addrFactory.NextUserDid()

	amount := uint64(1000)
	addMsg := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount,
		Period:    3600,
	}

	_, err := s.msgServer.AddUserSubscription(s.ctx, addMsg)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "not found")
}

func (s *UserSubscriptionTestSuite) TestUpdate() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	s.setupDeveloperWithLockup(developer, math.NewInt(2000))

	amount := uint64(1000)
	addMsg := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount,
		Period:    3600,
	}
	_, err := s.msgServer.AddUserSubscription(s.ctx, addMsg)
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
}

func (s *UserSubscriptionTestSuite) TestUpdateNonExistent() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	amount := uint64(1000)
	updateMsg := &tiertypes.MsgUpdateUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount,
		Period:    3600,
	}

	_, err := s.msgServer.UpdateUserSubscription(s.ctx, updateMsg)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "not subscribed")
}

func (s *UserSubscriptionTestSuite) TestRemove() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	s.setupDeveloperWithLockup(developer, math.NewInt(1000))

	amount := uint64(1000)
	addMsg := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    amount,
		Period:    3600,
	}
	_, err := s.msgServer.AddUserSubscription(s.ctx, addMsg)
	require.NoError(s.T(), err)

	removeMsg := &tiertypes.MsgRemoveUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
	}

	resp, err := s.msgServer.RemoveUserSubscription(s.ctx, removeMsg)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)

	sub := s.keeper.GetUserSubscription(s.ctx, developer, userDid)
	require.Nil(s.T(), sub)
}

func (s *UserSubscriptionTestSuite) TestRemoveNonExistent() {
	developer, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	removeMsg := &tiertypes.MsgRemoveUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
	}

	_, err := s.msgServer.RemoveUserSubscription(s.ctx, removeMsg)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "not subscribed")
}
