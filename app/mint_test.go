package app

import (
	"bytes"
	"context"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttestutil "github.com/cosmos/cosmos-sdk/x/mint/testutil"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	test "github.com/sourcenetwork/sourcehub/testutil"
	tierkeeper "github.com/sourcenetwork/sourcehub/x/tier/keeper"
)

type MintTestSuite struct {
	suite.Suite

	bankKeeper       *test.MockBankKeeper
	distrKeeper      *test.MockDistributionKeeper
	stakingKeeper    *test.MockStakingKeeper
	epochsKeeper     *test.MockEpochsKeeper
	tierKeeper       tierkeeper.Keeper
	mintKeeper       mintkeeper.Keeper
	queryClient      minttypes.QueryClient
	encCfg           test.EncodingConfig
	ctx              sdk.Context
	authorityAccount sdk.AccAddress
	logBuffer        *bytes.Buffer
	logger           log.Logger
}

func TestMintTestSuite(t *testing.T) {
	suite.Run(t, new(MintTestSuite))
}

func (suite *MintTestSuite) SetupTest() {
	sdkConfig := sdk.GetConfig()
	sdkConfig.SetBech32PrefixForAccount("source", "sourcepub")
	sdkConfig.SetBech32PrefixForValidator("sourcevaloper", "sourcevaloperpub")
	sdkConfig.SetBech32PrefixForConsensusNode("sourcevalcons", "sourcevalconspub")

	key := storetypes.NewKVStoreKey(minttypes.StoreKey)
	testCtx := testutil.DefaultContextWithDB(suite.T(), key, storetypes.NewTransientStoreKey("transient_test"))
	suite.ctx = testCtx.Ctx
	suite.encCfg = test.CreateTestEncodingConfig()
	suite.logBuffer = new(bytes.Buffer)
	suite.logger = log.NewLogger(suite.logBuffer)

	ctrl := gomock.NewController(suite.T())
	suite.bankKeeper = test.NewMockBankKeeper(ctrl)
	suite.distrKeeper = test.NewMockDistributionKeeper(ctrl)
	suite.stakingKeeper = test.NewMockStakingKeeper(ctrl)
	suite.epochsKeeper = test.NewMockEpochsKeeper(ctrl)
	suite.authorityAccount = sdk.AccAddress([]byte("authority"))

	accountKeeper := minttestutil.NewMockAccountKeeper(ctrl)
	accountKeeper.EXPECT().GetModuleAddress("mint").Return(sdk.AccAddress{})

	suite.tierKeeper = tierkeeper.NewKeeper(
		suite.encCfg.Codec,
		runtime.NewKVStoreService(key),
		log.NewNopLogger(),
		suite.authorityAccount.String(),
		suite.bankKeeper,
		suite.stakingKeeper,
		suite.epochsKeeper,
		suite.distrKeeper,
	)

	suite.mintKeeper = mintkeeper.NewKeeper(
		suite.encCfg.Codec,
		runtime.NewKVStoreService(key),
		suite.stakingKeeper,
		accountKeeper,
		suite.bankKeeper,
		authtypes.FeeCollectorName,
		suite.authorityAccount.String(),
	)

	err := suite.mintKeeper.Params.Set(suite.ctx, minttypes.DefaultParams())
	suite.Require().NoError(err)
	suite.Require().NoError(suite.mintKeeper.Minter.Set(suite.ctx, minttypes.DefaultInitialMinter()))
	queryHelper := baseapp.NewQueryServerTestHelper(suite.ctx, suite.encCfg.InterfaceRegistry)
	suite.queryClient = minttypes.NewQueryClient(queryHelper)

	defaultQueryServer := mintkeeper.NewQueryServerImpl(suite.mintKeeper)
	minttypes.RegisterQueryServer(queryHelper, NewCustomMintQueryServer(defaultQueryServer, suite.mintKeeper, &suite.tierKeeper, suite.logger))
}

func (suite *MintTestSuite) TestGRPCParams() {
	params, err := suite.queryClient.Params(context.Background(), &minttypes.QueryParamsRequest{})
	suite.Require().NoError(err)
	keeperParams, err := suite.mintKeeper.Params.Get(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(params.Params, keeperParams)

	totalBondedTokens := math.NewInt(1_000_000_000_000)
	suite.stakingKeeper.EXPECT().TotalBondedTokens(gomock.Any()).Return(totalBondedTokens, nil).Times(1)

	inflation, err := suite.queryClient.Inflation(context.Background(), &minttypes.QueryInflationRequest{})
	suite.Require().NoError(err)
	minter, err := suite.mintKeeper.Minter.Get(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(inflation.Inflation, minter.Inflation)

	annualProvisions, err := suite.queryClient.AnnualProvisions(context.Background(), &minttypes.QueryAnnualProvisionsRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal(annualProvisions.AnnualProvisions, minter.AnnualProvisions)
}

func (suite *MintTestSuite) TestInflationQuery() {
	minter, err := suite.mintKeeper.Minter.Get(suite.ctx)
	suite.Require().NoError(err)

	// Return default inflation rate if no bonded tokens
	totalBondedTokens := math.ZeroInt()
	suite.stakingKeeper.EXPECT().TotalBondedTokens(gomock.Any()).Return(totalBondedTokens, nil).Times(1)

	inflation, err := suite.queryClient.Inflation(context.Background(), &minttypes.QueryInflationRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal(inflation.Inflation, minter.Inflation)

	logs := suite.logBuffer.String()
	suite.Require().Contains(logs, "Returning default inflation")

	// Return effective inflation rate otherwise
	totalBondedTokens = math.NewInt(1_000_000_000_000)
	suite.stakingKeeper.EXPECT().TotalBondedTokens(gomock.Any()).Return(totalBondedTokens, nil).Times(1)

	inflation, err = suite.queryClient.Inflation(context.Background(), &minttypes.QueryInflationRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal(inflation.Inflation, minter.Inflation)

	logs = suite.logBuffer.String()
	suite.Require().Contains(logs, "Returning effective inflation")
}

func (suite *MintTestSuite) TestGetDelegatorStakeRatio() {
	delAddr, err := sdk.AccAddressFromBech32("source1wjj5v5rlf57kayyeskncpu4hwev25ty645p2et")
	suite.Require().NoError(err)
	valAddr, err := sdk.ValAddressFromBech32("sourcevaloper1cy0p47z24ejzvq55pu3lesxwf73xnrnd0pzkqm")
	suite.Require().NoError(err)

	testCases := []struct {
		desc          string
		totalStake    math.Int
		devStake      math.Int
		devPoolFee    int64
		insPoolFee    int64
		expectedRatio math.LegacyDec
		expectError   bool
	}{
		{
			desc:          "Zero dev stake",
			totalStake:    math.NewInt(1000),
			devStake:      math.ZeroInt(),
			devPoolFee:    2,
			insPoolFee:    1,
			expectedRatio: math.LegacyOneDec(),
			expectError:   false,
		},
		{
			desc:          "Existing dev stake",
			totalStake:    math.NewInt(1000),
			devStake:      math.NewInt(200),
			devPoolFee:    2,
			insPoolFee:    1,
			expectedRatio: math.LegacyMustNewDecFromStr("806").Quo(math.LegacyMustNewDecFromStr("1000")),
			expectError:   false,
		},
		{
			desc:          "Zero total stake",
			totalStake:    math.ZeroInt(),
			devStake:      math.NewInt(200),
			devPoolFee:    2,
			insPoolFee:    1,
			expectedRatio: math.LegacyOneDec(),
			expectError:   true,
		},
		{
			desc:          "Total stake less than dev stake",
			totalStake:    math.NewInt(100),
			devStake:      math.NewInt(200),
			devPoolFee:    2,
			insPoolFee:    1,
			expectedRatio: math.LegacyOneDec(),
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.desc, func() {
			suite.tierKeeper.AddLockup(suite.ctx, delAddr, valAddr, tc.devStake)
			params := suite.tierKeeper.GetParams(suite.ctx)
			params.DeveloperPoolFee = tc.devPoolFee
			params.InsurancePoolFee = tc.insPoolFee
			suite.tierKeeper.SetParams(suite.ctx, params)
			suite.stakingKeeper.EXPECT().TotalBondedTokens(gomock.Any()).Return(tc.totalStake, nil).Times(1)

			delStakeRatio, err := getDelegatorStakeRatio(suite.ctx, &suite.tierKeeper)

			if tc.expectError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expectedRatio, delStakeRatio)
			}
		})
	}
}
