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

type AutoLockIntegrationTestSuite struct {
	suite.Suite

	ctx         sdk.Context
	keeper      tierkeeper.Keeper
	msgServer   tiertypes.MsgServer
	addrFactory *TestAddressFactory
}

func (suite *AutoLockIntegrationTestSuite) SetupTest() {
	app.SetConfig(false)

	k, ctx := keepertest.TierKeeper(suite.T())
	suite.ctx = ctx
	suite.keeper = k
	suite.msgServer = tierkeeper.NewMsgServerImpl(&k)
	suite.addrFactory = NewTestAddressFactory()
}

func (suite *AutoLockIntegrationTestSuite) createTestAddresses() (developer, user, validator sdk.AccAddress) {
	developer, user = suite.addrFactory.NextPair(suite.T(), &suite.keeper, suite.ctx)
	validator = sdk.AccAddress(TestValidatorAddr)
	keepertest.CreateAccount(suite.T(), &suite.keeper, suite.ctx, validator)
	return developer, user, validator
}

func TestAutoLockIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(AutoLockIntegrationTestSuite))
}

func (s *AutoLockIntegrationTestSuite) TestAutoLockDisabledInsufficientCredits() {
	developer, _, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, math.NewInt(500))

	valAddr := sdk.ValAddress("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
	require.NoError(s.T(), err)

	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, math.NewInt(1000000))

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	addMsg := &tiertypes.MsgAddUserSubscription{
		Developer: developer.String(),
		UserDid:   userDid,
		Amount:    1000,
		Period:    3600,
	}

	_, err = s.msgServer.AddUserSubscription(s.ctx, addMsg)
	require.Error(s.T(), err)
	require.ErrorContains(s.T(), err, "insufficient credits and auto-lock disabled")
}

func (s *AutoLockIntegrationTestSuite) TestAutoLockEnabledWithLockup() {
	developer, _, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: true,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, math.NewInt(10000000))

	valAddr := sdk.ValAddress("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, math.NewInt(1000000))

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

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
	require.Equal(s.T(), amount, sub.CreditAmount)
}

func (s *AutoLockIntegrationTestSuite) TestAutoLockEnabledWithSufficientCredits() {
	developer, _, _ := s.createTestAddresses()
	userDid := s.addrFactory.NextUserDid()

	keepertest.InitializeDelegator(s.T(), &s.keeper, s.ctx, developer, math.NewInt(2000))

	valAddr, err := sdk.ValAddressFromBech32("veravaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnduc4vm0")
	require.NoError(s.T(), err)

	keepertest.InitializeValidator(s.T(), s.keeper.GetStakingKeeper().(*stakingkeeper.Keeper), s.ctx, valAddr, math.NewInt(1_000_000))

	s.ctx = s.ctx.WithBlockHeight(1).WithBlockTime(time.Now())

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: true,
	}

	_, err = s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

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
	require.Equal(s.T(), amount, sub.CreditAmount)
}

func (s *AutoLockIntegrationTestSuite) TestAutoLockToggling() {
	developer, _, _ := s.createTestAddresses()

	createMsg := &tiertypes.MsgCreateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: false,
	}
	_, err := s.msgServer.CreateDeveloper(s.ctx, createMsg)
	require.NoError(s.T(), err)

	dev := s.keeper.GetDeveloper(s.ctx, developer)
	require.NotNil(s.T(), dev)
	require.False(s.T(), dev.AutoLockEnabled)

	updateMsg := &tiertypes.MsgUpdateDeveloper{
		Developer:       developer.String(),
		AutoLockEnabled: true,
	}
	_, err = s.msgServer.UpdateDeveloper(s.ctx, updateMsg)
	require.NoError(s.T(), err)

	dev = s.keeper.GetDeveloper(s.ctx, developer)
	require.NotNil(s.T(), dev)
	require.True(s.T(), dev.AutoLockEnabled)

	updateMsg.AutoLockEnabled = false
	_, err = s.msgServer.UpdateDeveloper(s.ctx, updateMsg)
	require.NoError(s.T(), err)

	dev = s.keeper.GetDeveloper(s.ctx, developer)
	require.NotNil(s.T(), dev)
	require.False(s.T(), dev.AutoLockEnabled)
}
