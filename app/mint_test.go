package app

import (
	"bytes"
	gocontext "context"
	"testing"

	"github.com/golang/mock/gomock"
	tierkeeper "github.com/sourcenetwork/sourcehub/x/tier/keeper"
	"github.com/stretchr/testify/suite"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttestutil "github.com/cosmos/cosmos-sdk/x/mint/testutil"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	test "github.com/sourcenetwork/sourcehub/testutil"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
	minttypes.RegisterQueryServer(queryHelper, NewCustomMintQueryServer(defaultQueryServer, suite.mintKeeper, suite.tierKeeper, suite.logger))
}

func (suite *MintTestSuite) TestGRPCParams() {
	params, err := suite.queryClient.Params(gocontext.Background(), &minttypes.QueryParamsRequest{})
	suite.Require().NoError(err)
	keeperParams, err := suite.mintKeeper.Params.Get(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(params.Params, keeperParams)

	totalBondedTokens := math.NewInt(1_000_000_000_000)
	suite.stakingKeeper.EXPECT().TotalBondedTokens(gomock.Any()).Return(totalBondedTokens, nil).Times(1)

	inflation, err := suite.queryClient.Inflation(gocontext.Background(), &minttypes.QueryInflationRequest{})
	suite.Require().NoError(err)
	minter, err := suite.mintKeeper.Minter.Get(suite.ctx)
	suite.Require().NoError(err)
	suite.Require().Equal(inflation.Inflation, minter.Inflation)

	annualProvisions, err := suite.queryClient.AnnualProvisions(gocontext.Background(), &minttypes.QueryAnnualProvisionsRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal(annualProvisions.AnnualProvisions, minter.AnnualProvisions)
}

func (suite *MintTestSuite) TestInflationQuery() {
	minter, err := suite.mintKeeper.Minter.Get(suite.ctx)
	suite.Require().NoError(err)

	// Return default inflation rate if no bonded tokens
	totalBondedTokens := math.ZeroInt()
	suite.stakingKeeper.EXPECT().TotalBondedTokens(gomock.Any()).Return(totalBondedTokens, nil).Times(1)

	inflation, err := suite.queryClient.Inflation(gocontext.Background(), &minttypes.QueryInflationRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal(inflation.Inflation, minter.Inflation)

	logs := suite.logBuffer.String()
	suite.Require().Contains(logs, "Returning default inflation")

	// Return effective inflation rate otherwise
	totalBondedTokens = math.NewInt(1_000_000_000_000)
	suite.stakingKeeper.EXPECT().TotalBondedTokens(gomock.Any()).Return(totalBondedTokens, nil).Times(1)

	inflation, err = suite.queryClient.Inflation(gocontext.Background(), &minttypes.QueryInflationRequest{})
	suite.Require().NoError(err)
	suite.Require().Equal(inflation.Inflation, minter.Inflation)

	logs = suite.logBuffer.String()
	suite.Require().Contains(logs, "Returning effective inflation")
}
