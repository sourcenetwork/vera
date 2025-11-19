package keeper_test

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/sourcenetwork/sourcehub/x/feegrant"
	"github.com/sourcenetwork/sourcehub/x/feegrant/keeper"
	"github.com/sourcenetwork/sourcehub/x/feegrant/module"
	feegranttestutil "github.com/sourcenetwork/sourcehub/x/feegrant/testutil"

	codecaddress "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

type KeeperTestSuite struct {
	suite.Suite

	ctx            sdk.Context
	addrs          []sdk.AccAddress
	msgSrvr        feegrant.MsgServer
	coins          sdk.Coins
	feegrantKeeper keeper.Keeper
	accountKeeper  *feegranttestutil.MockAccountKeeper
	bankKeeper     *feegranttestutil.MockBankKeeper
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.addrs = simtestutil.CreateIncrementalAccounts(20)
	key := storetypes.NewKVStoreKey(feegrant.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModuleBasic{})

	// setup gomock and initialize some globally expected executions
	ctrl := gomock.NewController(suite.T())
	suite.accountKeeper = feegranttestutil.NewMockAccountKeeper(ctrl)
	for i := 0; i < len(suite.addrs); i++ {
		suite.accountKeeper.EXPECT().GetAccount(gomock.Any(), suite.addrs[i]).Return(authtypes.NewBaseAccountWithAddress(suite.addrs[i])).AnyTimes()
	}
	suite.accountKeeper.EXPECT().AddressCodec().Return(codecaddress.NewBech32Codec("cosmos")).AnyTimes()
	suite.bankKeeper = feegranttestutil.NewMockBankKeeper(ctrl)
	suite.bankKeeper.EXPECT().BlockedAddr(gomock.Any()).Return(false).AnyTimes()

	suite.feegrantKeeper = keeper.NewKeeper(encCfg.Codec, runtime.NewKVStoreService(key), suite.accountKeeper).SetBankKeeper(suite.bankKeeper)
	suite.ctx = testCtx.Ctx
	suite.msgSrvr = keeper.NewMsgServerImpl(suite.feegrantKeeper)
	suite.coins = sdk.NewCoins(sdk.NewCoin("uopen", sdkmath.NewInt(555)))
}

func (suite *KeeperTestSuite) TestKeeperCrud() {
	// some helpers
	eth := sdk.NewCoins(sdk.NewInt64Coin("eth", 123))
	exp := suite.ctx.BlockTime().AddDate(1, 0, 0)
	exp2 := suite.ctx.BlockTime().AddDate(2, 0, 0)
	basic := &feegrant.BasicAllowance{
		SpendLimit: suite.coins,
		Expiration: &exp,
	}

	basic2 := &feegrant.BasicAllowance{
		SpendLimit: eth,
		Expiration: &exp,
	}

	basic3 := &feegrant.BasicAllowance{
		SpendLimit: eth,
		Expiration: &exp2,
	}

	// let's set up some initial state here

	// addrs[0] -> addrs[1] (basic)
	err := suite.feegrantKeeper.GrantAllowance(suite.ctx, suite.addrs[0], suite.addrs[1], basic)
	suite.Require().NoError(err)

	// addrs[0] -> addrs[2] (basic2)
	err = suite.feegrantKeeper.GrantAllowance(suite.ctx, suite.addrs[0], suite.addrs[2], basic2)
	suite.Require().NoError(err)

	// addrs[1] -> addrs[2] (basic)
	err = suite.feegrantKeeper.GrantAllowance(suite.ctx, suite.addrs[1], suite.addrs[2], basic)
	suite.Require().NoError(err)

	// addrs[1] -> addrs[3] (basic)
	err = suite.feegrantKeeper.GrantAllowance(suite.ctx, suite.addrs[1], suite.addrs[3], basic)
	suite.Require().NoError(err)

	// addrs[3] -> addrs[0] (basic2)
	err = suite.feegrantKeeper.GrantAllowance(suite.ctx, suite.addrs[3], suite.addrs[0], basic2)
	suite.Require().NoError(err)

	// addrs[3] -> addrs[0] (basic2) expect error with duplicate grant
	err = suite.feegrantKeeper.GrantAllowance(suite.ctx, suite.addrs[3], suite.addrs[0], basic2)
	suite.Require().Error(err)

	// remove some, overwrite other
	_, err = suite.msgSrvr.RevokeAllowance(suite.ctx, &feegrant.MsgRevokeAllowance{Granter: suite.addrs[0].String(), Grantee: suite.addrs[1].String()})
	suite.Require().NoError(err)

	_, err = suite.msgSrvr.RevokeAllowance(suite.ctx, &feegrant.MsgRevokeAllowance{Granter: suite.addrs[0].String(), Grantee: suite.addrs[2].String()})
	suite.Require().NoError(err)

	// revoke non-exist fee allowance
	_, err = suite.msgSrvr.RevokeAllowance(suite.ctx, &feegrant.MsgRevokeAllowance{Granter: suite.addrs[0].String(), Grantee: suite.addrs[2].String()})
	suite.Require().Error(err)

	err = suite.feegrantKeeper.GrantAllowance(suite.ctx, suite.addrs[0], suite.addrs[2], basic)
	suite.Require().NoError(err)

	// revoke an existing grant and grant again with different allowance.
	_, err = suite.msgSrvr.RevokeAllowance(suite.ctx, &feegrant.MsgRevokeAllowance{Granter: suite.addrs[1].String(), Grantee: suite.addrs[2].String()})
	suite.Require().NoError(err)

	err = suite.feegrantKeeper.GrantAllowance(suite.ctx, suite.addrs[1], suite.addrs[2], basic3)
	suite.Require().NoError(err)

	// end state:
	// addr -> addr3 (basic)
	// addr2 -> addr3 (basic2), addr4(basic)
	// addr4 -> addr (basic2)

	// then lots of queries
	cases := map[string]struct {
		grantee   sdk.AccAddress
		granter   sdk.AccAddress
		allowance feegrant.FeeAllowanceI
	}{
		"addr revoked": {
			granter: suite.addrs[0],
			grantee: suite.addrs[1],
		},
		"addr revoked and added": {
			granter:   suite.addrs[0],
			grantee:   suite.addrs[2],
			allowance: basic,
		},
		"addr never there": {
			granter: suite.addrs[0],
			grantee: suite.addrs[3],
		},
		"addr modified": {
			granter:   suite.addrs[1],
			grantee:   suite.addrs[2],
			allowance: basic3,
		},
	}

	for name, tc := range cases {
		tc := tc
		suite.Run(name, func() {
			allow, _ := suite.feegrantKeeper.GetAllowance(suite.ctx, tc.granter, tc.grantee)

			if tc.allowance == nil {
				suite.Nil(allow)
				return
			}
			suite.NotNil(allow)
			suite.Equal(tc.allowance, allow)
		})
	}
	address := "cosmos1rxr4mq58w3gtnx5tsc438mwjjafv3mja7k5pnu"
	accAddr, err := codecaddress.NewBech32Codec("cosmos").StringToBytes(address)
	suite.Require().NoError(err)
	suite.accountKeeper.EXPECT().GetAccount(gomock.Any(), accAddr).Return(authtypes.NewBaseAccountWithAddress(accAddr)).AnyTimes()

	// let's grant and revoke authorization to non existing account
	err = suite.feegrantKeeper.GrantAllowance(suite.ctx, suite.addrs[3], accAddr, basic2)
	suite.Require().NoError(err)

	_, err = suite.feegrantKeeper.GetAllowance(suite.ctx, suite.addrs[3], accAddr)
	suite.Require().NoError(err)

	_, err = suite.msgSrvr.RevokeAllowance(suite.ctx, &feegrant.MsgRevokeAllowance{Granter: suite.addrs[3].String(), Grantee: address})
	suite.Require().NoError(err)
}

func (suite *KeeperTestSuite) TestUseGrantedFee() {
	eth := sdk.NewCoins(sdk.NewInt64Coin("eth", 123))
	blockTime := suite.ctx.BlockTime()
	oneYear := blockTime.AddDate(1, 0, 0)

	future := &feegrant.BasicAllowance{
		SpendLimit: suite.coins,
		Expiration: &oneYear,
	}

	// for testing limits of the contract
	hugeAmount := sdk.NewCoins(sdk.NewInt64Coin("uopen", 9999))
	smallAmount := sdk.NewCoins(sdk.NewInt64Coin("uopen", 1))
	futureAfterSmall := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 554)),
		Expiration: &oneYear,
	}

	// then lots of queries
	cases := map[string]struct {
		grantee sdk.AccAddress
		granter sdk.AccAddress
		fee     sdk.Coins
		allowed bool
		final   feegrant.FeeAllowanceI
		postRun func()
	}{
		"use entire pot": {
			granter: suite.addrs[0],
			grantee: suite.addrs[1],
			fee:     suite.coins,
			allowed: true,
			final:   nil,
			postRun: func() {},
		},
		"too high": {
			granter: suite.addrs[0],
			grantee: suite.addrs[1],
			fee:     hugeAmount,
			allowed: false,
			final:   future,
			postRun: func() {
				_, err := suite.msgSrvr.RevokeAllowance(suite.ctx, &feegrant.MsgRevokeAllowance{
					Granter: suite.addrs[0].String(),
					Grantee: suite.addrs[1].String(),
				})
				suite.Require().NoError(err)
			},
		},
		"use a little": {
			granter: suite.addrs[0],
			grantee: suite.addrs[1],
			fee:     smallAmount,
			allowed: true,
			final:   futureAfterSmall,
			postRun: func() {
				_, err := suite.msgSrvr.RevokeAllowance(suite.ctx, &feegrant.MsgRevokeAllowance{
					Granter: suite.addrs[0].String(),
					Grantee: suite.addrs[1].String(),
				})
				suite.Require().NoError(err)
			},
		},
	}

	for name, tc := range cases {
		tc := tc
		suite.Run(name, func() {
			err := suite.feegrantKeeper.GrantAllowance(suite.ctx, tc.granter, tc.grantee, future)
			suite.Require().NoError(err)

			err = suite.feegrantKeeper.UseGrantedFees(suite.ctx, tc.granter, tc.grantee, tc.fee, []sdk.Msg{})
			if tc.allowed {
				suite.NoError(err)
			} else {
				suite.Error(err)
			}

			loaded, _ := suite.feegrantKeeper.GetAllowance(suite.ctx, tc.granter, tc.grantee)
			suite.Equal(tc.final, loaded)

			tc.postRun()
		})
	}

	basicAllowance := &feegrant.BasicAllowance{
		SpendLimit: eth,
		Expiration: &blockTime,
	}

	// create basic fee allowance
	err := suite.feegrantKeeper.GrantAllowance(suite.ctx, suite.addrs[0], suite.addrs[2], basicAllowance)
	suite.Require().NoError(err)

	// waiting for future blocks, allowance to be pruned.
	ctx := suite.ctx.WithBlockTime(oneYear)

	// expect error: feegrant expired
	err = suite.feegrantKeeper.UseGrantedFees(ctx, suite.addrs[0], suite.addrs[2], eth, []sdk.Msg{})
	suite.Error(err)
	suite.Contains(err.Error(), "fee allowance expired")

	// verify: feegrant is revoked
	_, err = suite.feegrantKeeper.GetAllowance(ctx, suite.addrs[0], suite.addrs[2])
	suite.Error(err)
	suite.Contains(err.Error(), "fee-grant not found")
}

func (suite *KeeperTestSuite) TestIterateGrants() {
	eth := sdk.NewCoins(sdk.NewInt64Coin("eth", 123))
	exp := suite.ctx.BlockTime().AddDate(1, 0, 0)

	allowance := &feegrant.BasicAllowance{
		SpendLimit: suite.coins,
		Expiration: &exp,
	}

	allowance1 := &feegrant.BasicAllowance{
		SpendLimit: eth,
		Expiration: &exp,
	}

	suite.feegrantKeeper.GrantAllowance(suite.ctx, suite.addrs[0], suite.addrs[1], allowance)
	suite.feegrantKeeper.GrantAllowance(suite.ctx, suite.addrs[2], suite.addrs[1], allowance1)

	suite.feegrantKeeper.IterateAllFeeAllowances(suite.ctx, func(grant feegrant.Grant) bool {
		suite.Require().Equal(suite.addrs[1].String(), grant.Grantee)
		suite.Require().Contains([]string{suite.addrs[0].String(), suite.addrs[2].String()}, grant.Granter)
		return true
	})
}

func (suite *KeeperTestSuite) TestPruneGrants() {
	eth := sdk.NewCoins(sdk.NewInt64Coin("eth", 123))
	now := suite.ctx.BlockTime()
	oneYearExpiry := now.AddDate(1, 0, 0)

	testCases := []struct {
		name      string
		ctx       sdk.Context
		granter   sdk.AccAddress
		grantee   sdk.AccAddress
		allowance feegrant.FeeAllowanceI
		expErrMsg string
		preRun    func()
		postRun   func()
	}{
		{
			name:    "grant not pruned from state",
			ctx:     suite.ctx,
			granter: suite.addrs[0],
			grantee: suite.addrs[1],
			allowance: &feegrant.BasicAllowance{
				SpendLimit: suite.coins,
				Expiration: &now,
			},
		},
		{
			name:      "grant pruned from state after a block: error",
			ctx:       suite.ctx.WithBlockTime(now.AddDate(0, 0, 1)),
			granter:   suite.addrs[2],
			grantee:   suite.addrs[1],
			expErrMsg: "not found",
			allowance: &feegrant.BasicAllowance{
				SpendLimit: eth,
				Expiration: &now,
			},
		},
		{
			name:    "grant not pruned from state after a day: no error",
			ctx:     suite.ctx.WithBlockTime(now.AddDate(0, 0, 1)),
			granter: suite.addrs[1],
			grantee: suite.addrs[0],
			allowance: &feegrant.BasicAllowance{
				SpendLimit: eth,
				Expiration: &oneYearExpiry,
			},
		},
		{
			name:      "grant pruned from state after a year: error",
			ctx:       suite.ctx.WithBlockTime(now.AddDate(1, 0, 1)),
			granter:   suite.addrs[1],
			grantee:   suite.addrs[2],
			expErrMsg: "not found",
			allowance: &feegrant.BasicAllowance{
				SpendLimit: eth,
				Expiration: &oneYearExpiry,
			},
		},
		{
			name:    "no expiry: no error",
			ctx:     suite.ctx.WithBlockTime(now.AddDate(1, 0, 0)),
			granter: suite.addrs[1],
			grantee: suite.addrs[2],
			allowance: &feegrant.BasicAllowance{
				SpendLimit: eth,
				Expiration: &oneYearExpiry,
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			if tc.preRun != nil {
				tc.preRun()
			}
			err := suite.feegrantKeeper.GrantAllowance(suite.ctx, tc.granter, tc.grantee, tc.allowance)
			suite.NoError(err)
			err = suite.feegrantKeeper.RemoveExpiredAllowances(tc.ctx, 5)
			suite.NoError(err)

			grant, err := suite.feegrantKeeper.GetAllowance(tc.ctx, tc.granter, tc.grantee)
			if tc.expErrMsg != "" {
				suite.Error(err)
				suite.Contains(err.Error(), tc.expErrMsg)
			} else {
				suite.NotNil(grant)
			}
			if tc.postRun != nil {
				tc.postRun()
			}
		})
	}
}

func (suite *KeeperTestSuite) TestDIDAllowanceGrant() {
	testDID := "did:example:bob"

	testCases := []struct {
		name          string
		granter       sdk.AccAddress
		granteeDID    string
		allowance     feegrant.FeeAllowanceI
		expectedError string
		preRun        func()
	}{
		{
			name:       "valid DID grant",
			granter:    suite.addrs[0],
			granteeDID: testDID,
			allowance: &feegrant.BasicAllowance{
				SpendLimit: suite.coins,
			},
		},
		{
			name:       "duplicate DID grant",
			granter:    suite.addrs[1],
			granteeDID: testDID,
			allowance:  &feegrant.BasicAllowance{SpendLimit: suite.coins},
			preRun: func() {
				// Grant first time
				err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, suite.addrs[1], testDID, &feegrant.BasicAllowance{SpendLimit: suite.coins})
				suite.NoError(err)
			},
			expectedError: "already exists",
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			if tc.preRun != nil {
				tc.preRun()
			}

			err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, tc.granter, tc.granteeDID, tc.allowance)

			if tc.expectedError != "" {
				suite.Error(err)
				suite.Contains(err.Error(), tc.expectedError)
				return
			}

			suite.NoError(err)

			// Verify the grant was created
			allowance, err := suite.feegrantKeeper.GetDIDAllowance(suite.ctx, tc.granter, tc.granteeDID)
			suite.NoError(err)
			suite.NotNil(allowance)
		})
	}
}

func (suite *KeeperTestSuite) TestDIDAllowanceRevoke() {
	testDID := "did:example:alice"
	granter := suite.addrs[2]

	now := sdk.UnwrapSDKContext(suite.ctx).BlockTime()
	spendLimit := sdk.NewCoins(sdk.NewCoin(appparams.MicroCreditDenom, math.NewInt(100)))
	period := time.Hour

	periodicAllowance := &feegrant.PeriodicAllowance{
		Basic: feegrant.BasicAllowance{
			SpendLimit: suite.coins,
		},
		Period:           period,
		PeriodSpendLimit: spendLimit,
		PeriodCanSpend:   spendLimit,
		PeriodReset:      now.Add(period),
	}

	// First grant a periodic allowance
	err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter, testDID, periodicAllowance)
	suite.NoError(err)

	// Verify it exists
	existingAllowance, err := suite.feegrantKeeper.GetDIDAllowance(suite.ctx, granter, testDID)
	suite.NoError(err)
	suite.NotNil(existingAllowance)

	// Verify there was no expiration
	existingExpiration, err := existingAllowance.ExpiresAt()
	suite.NoError(err)
	suite.Nil(existingExpiration)

	// Now expire it
	err = suite.feegrantKeeper.ExpireDIDAllowance(suite.ctx, granter, testDID)
	suite.NoError(err)

	// Verify that allowance exists and has expiration set
	expiredAllowance, err := suite.feegrantKeeper.GetDIDAllowance(suite.ctx, granter, testDID)
	suite.NoError(err)
	suite.NotNil(expiredAllowance)

	expiredExpiration, err := expiredAllowance.ExpiresAt()
	suite.NotNil(expiredExpiration)
}

func (suite *KeeperTestSuite) TestDIDAllowanceUsage() {
	testDID := "did:example:bob"
	granter := suite.addrs[3]

	// Create allowance with specific spend limit
	spendLimit := sdk.NewCoins(sdk.NewInt64Coin("uopen", 1000))
	allowance := &feegrant.BasicAllowance{SpendLimit: spendLimit}

	err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter, testDID, allowance)
	suite.NoError(err)

	// Test fee usage
	fee := sdk.NewCoins(sdk.NewInt64Coin("uopen", 100))
	msgs := []sdk.Msg{}

	err = suite.feegrantKeeper.UseGrantedFeesByDID(suite.ctx, granter, testDID, fee, msgs)
	suite.NoError(err)

	// Check remaining allowance
	remainingAllowance, err := suite.feegrantKeeper.GetDIDAllowance(suite.ctx, granter, testDID)
	suite.NoError(err)

	basic, ok := remainingAllowance.(*feegrant.BasicAllowance)
	suite.True(ok)
	expected := sdk.NewCoins(sdk.NewInt64Coin("uopen", 900))
	suite.Equal(expected, basic.SpendLimit)
}

func (suite *KeeperTestSuite) TestDIDAllowanceWithFallback() {
	testDID := "did:example:alice"
	granter := suite.addrs[4]

	// Create DID allowance
	didAllowance := &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 500)),
	}
	err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter, testDID, didAllowance)
	suite.NoError(err)

	fee := sdk.NewCoins(sdk.NewInt64Coin("uopen", 100))
	msgs := []sdk.Msg{} // Mock messages with DID

	err = suite.feegrantKeeper.UseGrantedFeesByDID(suite.ctx, granter, testDID, fee, msgs)
	suite.NoError(err)
}

func (suite *KeeperTestSuite) TestDIDAllowanceExpiry() {
	testDID := "did:example:bob"
	granter := suite.addrs[6]

	// Create allowance that expires in future
	now := suite.ctx.BlockTime()
	expiry := now.Add(time.Hour) // Expires in 1 hour

	allowance := &feegrant.BasicAllowance{
		SpendLimit: suite.coins,
		Expiration: &expiry,
	}

	err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter, testDID, allowance)
	suite.NoError(err)

	// Advance time to make allowance expire
	expiredCtx := suite.ctx.WithBlockTime(now.Add(2 * time.Hour))
	suite.ctx = expiredCtx

	// Try to use expired allowance
	fee := sdk.NewCoins(sdk.NewInt64Coin("uopen", 10))
	msgs := []sdk.Msg{}

	err = suite.feegrantKeeper.UseGrantedFeesByDID(suite.ctx, granter, testDID, fee, msgs)
	suite.Error(err)
	suite.Contains(err.Error(), "expired")

	// Verify allowance was removed due to expiry
	_, err = suite.feegrantKeeper.GetDIDAllowance(suite.ctx, granter, testDID)
	suite.Error(err)
	suite.Contains(err.Error(), "not found")
}

// TestSeparateQueueSystems verifies that DID grants and regular grants use completely separate expiration queue systems
func (suite *KeeperTestSuite) TestSeparateQueueSystems() {
	now := suite.ctx.BlockTime()
	expiry := now.Add(time.Hour) // Both types expire in 1 hour

	// Create regular grant (uses 0x01 queue prefix)
	regularGranter := suite.addrs[0]
	regularGrantee := suite.addrs[1]
	regularAllowance := &feegrant.BasicAllowance{
		SpendLimit: suite.coins,
		Expiration: &expiry,
	}
	err := suite.feegrantKeeper.GrantAllowance(suite.ctx, regularGranter, regularGrantee, regularAllowance)
	suite.NoError(err)

	// Create DID grant (uses 0x03 queue prefix)
	didGranter := suite.addrs[2]
	testDID := "did:example:alice"
	didAllowance := &feegrant.BasicAllowance{
		SpendLimit: suite.coins,
		Expiration: &expiry,
	}
	err = suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, didGranter, testDID, didAllowance)
	suite.NoError(err)

	// Verify both grants exist before expiry
	_, err = suite.feegrantKeeper.GetAllowance(suite.ctx, regularGranter, regularGrantee)
	suite.NoError(err)
	_, err = suite.feegrantKeeper.GetDIDAllowance(suite.ctx, didGranter, testDID)
	suite.NoError(err)

	// Advance time to expire both grants
	expiredCtx := suite.ctx.WithBlockTime(now.Add(2 * time.Hour))

	// Remove expired regular grants only - this should NOT affect DID grants
	err = suite.feegrantKeeper.RemoveExpiredAllowances(expiredCtx, 10)
	suite.NoError(err)

	// Regular grant should be removed
	_, err = suite.feegrantKeeper.GetAllowance(expiredCtx, regularGranter, regularGrantee)
	suite.Error(err)
	suite.Contains(err.Error(), "not found")

	// DID grant should still exist (not removed by regular grant cleanup)
	_, err = suite.feegrantKeeper.GetDIDAllowance(expiredCtx, didGranter, testDID)
	suite.NoError(err)

	// Now remove expired DID grants only
	err = suite.feegrantKeeper.RemoveExpiredDIDAllowances(expiredCtx, 10)
	suite.NoError(err)

	// Now DID grant should be removed
	_, err = suite.feegrantKeeper.GetDIDAllowance(expiredCtx, didGranter, testDID)
	suite.Error(err)
	suite.Contains(err.Error(), "not found")
}

// TestQueueKeyPrefixSeparation tests that regular and DID grants use different key prefixes in their queues
func (suite *KeeperTestSuite) TestQueueKeyPrefixSeparation() {
	// Test the key prefixes are different
	suite.NotEqual(feegrant.FeeAllowanceQueueKeyPrefix, feegrant.DIDFeeAllowanceQueueKeyPrefix)

	// Regular grants use 0x01 prefix
	suite.Equal([]byte{0x01}, feegrant.FeeAllowanceQueueKeyPrefix)

	// DID grants use 0x03 prefix
	suite.Equal([]byte{0x03}, feegrant.DIDFeeAllowanceQueueKeyPrefix)

	// Test storage key prefixes are also different
	suite.NotEqual(feegrant.FeeAllowanceKeyPrefix, feegrant.DIDFeeAllowanceKeyPrefix)

	// Regular grants use 0x00 prefix
	suite.Equal([]byte{0x00}, feegrant.FeeAllowanceKeyPrefix)

	// DID grants use 0x02 prefix
	suite.Equal([]byte{0x02}, feegrant.DIDFeeAllowanceKeyPrefix)
}

// TestMixedExpirationHandling tests that mixed regular and DID grants with different expiration times are handled correctly
func (suite *KeeperTestSuite) TestMixedExpirationHandling() {
	now := suite.ctx.BlockTime()
	shortExpiry := now.Add(time.Hour)
	longExpiry := now.Add(24 * time.Hour)

	// Create regular grants with different expiry times
	regularGranter1 := suite.addrs[0]
	regularGrantee1 := suite.addrs[1]
	regularAllowanceShort := &feegrant.BasicAllowance{
		SpendLimit: suite.coins,
		Expiration: &shortExpiry,
	}
	err := suite.feegrantKeeper.GrantAllowance(suite.ctx, regularGranter1, regularGrantee1, regularAllowanceShort)
	suite.NoError(err)

	regularGranter2 := suite.addrs[2]
	regularGrantee2 := suite.addrs[3]
	regularAllowanceLong := &feegrant.BasicAllowance{
		SpendLimit: suite.coins,
		Expiration: &longExpiry,
	}
	err = suite.feegrantKeeper.GrantAllowance(suite.ctx, regularGranter2, regularGrantee2, regularAllowanceLong)
	suite.NoError(err)

	// Create DID grants with different expiry times
	didGranter1 := suite.addrs[4]
	testDID1 := "did:example:alice"
	didAllowanceShort := &feegrant.BasicAllowance{
		SpendLimit: suite.coins,
		Expiration: &shortExpiry,
	}
	err = suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, didGranter1, testDID1, didAllowanceShort)
	suite.NoError(err)

	didGranter2 := suite.addrs[5]
	testDID2 := "did:example:bob"
	didAllowanceLong := &feegrant.BasicAllowance{
		SpendLimit: suite.coins,
		Expiration: &longExpiry,
	}
	err = suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, didGranter2, testDID2, didAllowanceLong)
	suite.NoError(err)

	// Advance time to expire short-term grants but not long-term
	midCtx := suite.ctx.WithBlockTime(now.Add(2 * time.Hour))

	// Remove expired regular grants
	err = suite.feegrantKeeper.RemoveExpiredAllowances(midCtx, 10)
	suite.NoError(err)

	// Short-term regular grant should be removed
	_, err = suite.feegrantKeeper.GetAllowance(midCtx, regularGranter1, regularGrantee1)
	suite.Error(err)

	// Long-term regular grant should still exist
	_, err = suite.feegrantKeeper.GetAllowance(midCtx, regularGranter2, regularGrantee2)
	suite.NoError(err)

	// DID grants should not be affected by regular grant cleanup
	_, err = suite.feegrantKeeper.GetDIDAllowance(midCtx, didGranter1, testDID1)
	suite.NoError(err)
	_, err = suite.feegrantKeeper.GetDIDAllowance(midCtx, didGranter2, testDID2)
	suite.NoError(err)

	// Now remove expired DID grants
	err = suite.feegrantKeeper.RemoveExpiredDIDAllowances(midCtx, 10)
	suite.NoError(err)

	// Short-term DID grant should be removed
	_, err = suite.feegrantKeeper.GetDIDAllowance(midCtx, didGranter1, testDID1)
	suite.Error(err)

	// Long-term DID grant should still exist
	_, err = suite.feegrantKeeper.GetDIDAllowance(midCtx, didGranter2, testDID2)
	suite.NoError(err)

	// Regular grants should not be affected by DID grant cleanup
	_, err = suite.feegrantKeeper.GetAllowance(midCtx, regularGranter2, regularGrantee2)
	suite.NoError(err)
}

// TestIterationSeparation tests that iteration functions only return their respective grant types
func (suite *KeeperTestSuite) TestIterationSeparation() {
	// Create mixed grants
	regularGranter := suite.addrs[0]
	regularGrantee := suite.addrs[1]
	regularAllowance := &feegrant.BasicAllowance{SpendLimit: suite.coins}
	err := suite.feegrantKeeper.GrantAllowance(suite.ctx, regularGranter, regularGrantee, regularAllowance)
	suite.NoError(err)

	didGranter := suite.addrs[2]
	testDID := "did:example:alice"
	didAllowance := &feegrant.BasicAllowance{SpendLimit: suite.coins}
	err = suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, didGranter, testDID, didAllowance)
	suite.NoError(err)

	// Test regular grant iteration only finds regular grants
	regularCount := 0
	err = suite.feegrantKeeper.IterateAllFeeAllowances(suite.ctx, func(grant feegrant.Grant) bool {
		regularCount++
		// Should be regular grant format - grantee is an address
		suite.Equal(regularGrantee.String(), grant.Grantee)
		suite.Equal(regularGranter.String(), grant.Granter)
		return true
	})
	suite.NoError(err)
	suite.Equal(1, regularCount)

	// Test DID grant iteration only finds DID grants
	didCount := 0
	err = suite.feegrantKeeper.IterateAllDIDAllowances(suite.ctx, func(grant feegrant.DIDGrant) bool {
		didCount++
		// Should be DID grant format - grantee is a DID
		suite.Equal(testDID, grant.GranteeDid)
		suite.Equal(didGranter.String(), grant.Granter)
		return true
	})
	suite.NoError(err)
	suite.Equal(1, didCount)
}

// TestGetFirstAvailableDIDGrant tests retrieving the first available grant for a DID
func (suite *KeeperTestSuite) TestGetFirstAvailableDIDGrant() {
	testDID := "did:example:charlie"

	testCases := []struct {
		name          string
		setupGrants   func()
		expectedError string
		validateGrant func(granter sdk.AccAddress, allowance feegrant.FeeAllowanceI)
	}{
		{
			name: "no grants available",
			setupGrants: func() {
				// Don't create any grants
			},
			expectedError: "no fee-grant found for DID",
		},
		{
			name: "single grant available",
			setupGrants: func() {
				granter := suite.addrs[10]
				allowance := &feegrant.BasicAllowance{
					SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 1000)),
				}
				err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter, testDID, allowance)
				suite.NoError(err)
			},
			validateGrant: func(granter sdk.AccAddress, allowance feegrant.FeeAllowanceI) {
				suite.Equal(suite.addrs[10], granter)
				basic, ok := allowance.(*feegrant.BasicAllowance)
				suite.True(ok)
				suite.Equal(sdk.NewCoins(sdk.NewInt64Coin("uopen", 1000)), basic.SpendLimit)
			},
		},
		{
			name: "multiple grants available - returns first one",
			setupGrants: func() {
				// Create multiple grants for the same DID
				granter1 := suite.addrs[11]
				allowance1 := &feegrant.BasicAllowance{
					SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 500)),
				}
				err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter1, testDID, allowance1)
				suite.NoError(err)

				granter2 := suite.addrs[12]
				allowance2 := &feegrant.BasicAllowance{
					SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 2000)),
				}
				err = suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter2, testDID, allowance2)
				suite.NoError(err)
			},
			validateGrant: func(granter sdk.AccAddress, allowance feegrant.FeeAllowanceI) {
				// Should return one of the grants
				suite.NotNil(granter)
				suite.NotNil(allowance)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			if tc.setupGrants != nil {
				tc.setupGrants()
			}

			granter, allowance, err := suite.feegrantKeeper.GetFirstAvailableDIDGrant(suite.ctx, testDID)

			if tc.expectedError != "" {
				suite.Error(err)
				suite.Contains(err.Error(), tc.expectedError)
				suite.Nil(granter)
				suite.Nil(allowance)
				return
			}

			suite.NoError(err)
			if tc.validateGrant != nil {
				tc.validateGrant(granter, allowance)
			}
		})
	}
}

// TestUseFirstAvailableDIDGrant tests using the first available grant for a DID
func (suite *KeeperTestSuite) TestUseFirstAvailableDIDGrant() {
	testDID := "did:example:dave"

	testCases := []struct {
		name          string
		setupGrants   func()
		fee           sdk.Coins
		expectedError string
		validateAfter func()
	}{
		{
			name: "no grants available",
			setupGrants: func() {
				// Don't create any grants
			},
			fee:           sdk.NewCoins(sdk.NewInt64Coin("uopen", 100)),
			expectedError: "no usable fee-grant found for DID",
		},
		{
			name: "single grant - successful use",
			setupGrants: func() {
				granter := suite.addrs[13]
				allowance := &feegrant.BasicAllowance{
					SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 1000)),
				}
				err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter, testDID, allowance)
				suite.NoError(err)
			},
			fee: sdk.NewCoins(sdk.NewInt64Coin("uopen", 100)),
			validateAfter: func() {
				// Grant should still exist with reduced spend limit
				allowance, err := suite.feegrantKeeper.GetDIDAllowance(suite.ctx, suite.addrs[13], testDID)
				suite.NoError(err)
				basic, ok := allowance.(*feegrant.BasicAllowance)
				suite.True(ok)
				suite.Equal(sdk.NewCoins(sdk.NewInt64Coin("uopen", 900)), basic.SpendLimit)
			},
		},
		{
			name: "single grant - depletes allowance completely",
			setupGrants: func() {
				granter := suite.addrs[14]
				allowance := &feegrant.BasicAllowance{
					SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 100)),
				}
				err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter, testDID, allowance)
				suite.NoError(err)
			},
			fee: sdk.NewCoins(sdk.NewInt64Coin("uopen", 100)),
			validateAfter: func() {
				// Grant should be removed when fully depleted
				_, err := suite.feegrantKeeper.GetDIDAllowance(suite.ctx, suite.addrs[14], testDID)
				suite.Error(err)
				suite.Contains(err.Error(), "not found")
			},
		},
		{
			name: "single grant - fee exceeds limit",
			setupGrants: func() {
				granter := suite.addrs[15]
				allowance := &feegrant.BasicAllowance{
					SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 50)),
				}
				err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter, testDID, allowance)
				suite.NoError(err)
			},
			fee:           sdk.NewCoins(sdk.NewInt64Coin("uopen", 100)),
			expectedError: "no usable fee-grant found for DID",
		},
		{
			name: "multiple grants - first one insufficient, second one works",
			setupGrants: func() {
				// First grant with insufficient limit
				granter1 := suite.addrs[16]
				allowance1 := &feegrant.BasicAllowance{
					SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 50)),
				}
				err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter1, testDID, allowance1)
				suite.NoError(err)

				// Second grant with sufficient limit
				granter2 := suite.addrs[17]
				allowance2 := &feegrant.BasicAllowance{
					SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 1000)),
				}
				err = suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter2, testDID, allowance2)
				suite.NoError(err)
			},
			fee: sdk.NewCoins(sdk.NewInt64Coin("uopen", 100)),
			validateAfter: func() {
				// First grant should still exist (wasn't used)
				allowance1, err := suite.feegrantKeeper.GetDIDAllowance(suite.ctx, suite.addrs[16], testDID)
				suite.NoError(err)
				basic1, ok := allowance1.(*feegrant.BasicAllowance)
				suite.True(ok)
				suite.Equal(sdk.NewCoins(sdk.NewInt64Coin("uopen", 50)), basic1.SpendLimit)

				// Second grant should have reduced limit
				allowance2, err := suite.feegrantKeeper.GetDIDAllowance(suite.ctx, suite.addrs[17], testDID)
				suite.NoError(err)
				basic2, ok := allowance2.(*feegrant.BasicAllowance)
				suite.True(ok)
				suite.Equal(sdk.NewCoins(sdk.NewInt64Coin("uopen", 900)), basic2.SpendLimit)
			},
		},
		{
			name: "expired grant - should be skipped and removed",
			setupGrants: func() {
				now := suite.ctx.BlockTime()
				futureExpiry := now.Add(time.Hour)

				granter := suite.addrs[18]
				allowance := &feegrant.BasicAllowance{
					SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 1000)),
					Expiration: &futureExpiry,
				}
				err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter, testDID, allowance)
				suite.NoError(err)

				// Advance time to make it expired
				suite.ctx = suite.ctx.WithBlockTime(now.Add(2 * time.Hour))
			},
			fee:           sdk.NewCoins(sdk.NewInt64Coin("uopen", 100)),
			expectedError: "no usable fee-grant found for DID",
			validateAfter: func() {
				// Expired grant should be removed
				_, err := suite.feegrantKeeper.GetDIDAllowance(suite.ctx, suite.addrs[18], testDID)
				suite.Error(err)
				suite.Contains(err.Error(), "not found")
			},
		},
		{
			name: "mixed grants - expired first, valid second",
			setupGrants: func() {
				now := suite.ctx.BlockTime()
				futureExpiry := now.Add(time.Hour)

				// First grant - will be expired (using lower index so it comes first in iteration)
				granter1 := suite.addrs[7]
				allowance1 := &feegrant.BasicAllowance{
					SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 1000)),
					Expiration: &futureExpiry,
				}
				err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter1, testDID, allowance1)
				suite.NoError(err)

				// Second grant - valid (no expiration, using higher index so it comes second in iteration)
				granter2 := suite.addrs[19]
				allowance2 := &feegrant.BasicAllowance{
					SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 1000)),
				}
				err = suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter2, testDID, allowance2)
				suite.NoError(err)

				// Advance time to make first grant expired
				suite.ctx = suite.ctx.WithBlockTime(now.Add(2 * time.Hour))
			},
			fee: sdk.NewCoins(sdk.NewInt64Coin("uopen", 100)),
			validateAfter: func() {
				// First grant should be removed (expired)
				_, err := suite.feegrantKeeper.GetDIDAllowance(suite.ctx, suite.addrs[7], testDID)
				suite.Error(err)
				suite.Contains(err.Error(), "not found")

				// Second grant should have reduced limit
				allowance2, err := suite.feegrantKeeper.GetDIDAllowance(suite.ctx, suite.addrs[19], testDID)
				suite.NoError(err)
				basic2, ok := allowance2.(*feegrant.BasicAllowance)
				suite.True(ok)
				suite.Equal(sdk.NewCoins(sdk.NewInt64Coin("uopen", 900)), basic2.SpendLimit)
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest()

			if tc.setupGrants != nil {
				tc.setupGrants()
			}

			granter, err := suite.feegrantKeeper.UseFirstAvailableDIDGrant(suite.ctx, testDID, tc.fee, []sdk.Msg{})

			if tc.expectedError != "" {
				suite.Error(err)
				suite.Contains(err.Error(), tc.expectedError)
				suite.Nil(granter)
				return
			}

			suite.NoError(err)
			suite.NotNil(granter)

			if tc.validateAfter != nil {
				tc.validateAfter()
			}
		})
	}
}
