package keeper_test

import (
	"github.com/sourcenetwork/sourcehub/x/feegrant"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	invalidGrantee = "invalid-grantee"
	invalidGranter = "invalid-granter"
)

func (suite *KeeperTestSuite) TestFeeAllowance() {
	testCases := []struct {
		name      string
		req       *feegrant.QueryAllowanceRequest
		expectErr bool
		preRun    func()
		postRun   func(_ *feegrant.QueryAllowanceResponse)
	}{
		{
			"nil request",
			nil,
			true,
			func() {},
			func(*feegrant.QueryAllowanceResponse) {},
		},
		{
			"fail: invalid granter",
			&feegrant.QueryAllowanceRequest{
				Granter: invalidGranter,
				Grantee: suite.addrs[0].String(),
			},
			true,
			func() {},
			func(*feegrant.QueryAllowanceResponse) {},
		},
		{
			"fail: invalid grantee",
			&feegrant.QueryAllowanceRequest{
				Granter: suite.addrs[0].String(),
				Grantee: invalidGrantee,
			},
			true,
			func() {},
			func(*feegrant.QueryAllowanceResponse) {},
		},
		{
			"fail: no grants",
			&feegrant.QueryAllowanceRequest{
				Granter: suite.addrs[0].String(),
				Grantee: suite.addrs[1].String(),
			},
			true,
			func() {},
			func(*feegrant.QueryAllowanceResponse) {},
		},
		{
			"non existed grant",
			&feegrant.QueryAllowanceRequest{
				Granter: invalidGranter,
				Grantee: invalidGrantee,
			},
			true,
			func() {},
			func(*feegrant.QueryAllowanceResponse) {},
		},
		{
			"valid query: expect single grant",
			&feegrant.QueryAllowanceRequest{
				Granter: suite.addrs[0].String(),
				Grantee: suite.addrs[1].String(),
			},
			false,
			func() {
				suite.grantFeeAllowance(suite.addrs[0], suite.addrs[1])
			},
			func(response *feegrant.QueryAllowanceResponse) {
				suite.Require().Equal(response.Allowance.Granter, suite.addrs[0].String())
				suite.Require().Equal(response.Allowance.Grantee, suite.addrs[1].String())
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.preRun()
			resp, err := suite.feegrantKeeper.Allowance(suite.ctx, tc.req)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				tc.postRun(resp)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestMissingAllowancesReturnNotFound() {
	_, err := suite.feegrantKeeper.Allowance(suite.ctx, &feegrant.QueryAllowanceRequest{
		Granter: suite.addrs[18].String(),
		Grantee: suite.addrs[19].String(),
	})
	suite.Require().Equal(codes.NotFound, status.Code(err))

	_, err = suite.feegrantKeeper.DIDAllowance(suite.ctx, &feegrant.QueryDIDAllowanceRequest{
		Granter:    suite.addrs[18].String(),
		GranteeDid: "did:example:missing",
	})
	suite.Require().Equal(codes.NotFound, status.Code(err))
}

func (suite *KeeperTestSuite) TestFeeAllowances() {
	testCases := []struct {
		name      string
		req       *feegrant.QueryAllowancesRequest
		expectErr bool
		preRun    func()
		postRun   func(_ *feegrant.QueryAllowancesResponse)
	}{
		{
			"nil request",
			nil,
			true,
			func() {},
			func(*feegrant.QueryAllowancesResponse) {},
		},
		{
			"fail: invalid grantee",
			&feegrant.QueryAllowancesRequest{
				Grantee: invalidGrantee,
			},
			true,
			func() {},
			func(*feegrant.QueryAllowancesResponse) {},
		},
		{
			"no grants",
			&feegrant.QueryAllowancesRequest{
				Grantee: suite.addrs[1].String(),
			},
			false,
			func() {},
			func(resp *feegrant.QueryAllowancesResponse) {
				suite.Require().Equal(len(resp.Allowances), 0)
			},
		},
		{
			"valid query: expect single grant",
			&feegrant.QueryAllowancesRequest{
				Grantee: suite.addrs[1].String(),
			},
			false,
			func() {
				suite.grantFeeAllowance(suite.addrs[0], suite.addrs[1])
			},
			func(resp *feegrant.QueryAllowancesResponse) {
				suite.Require().Equal(len(resp.Allowances), 1)
				suite.Require().Equal(resp.Allowances[0].Granter, suite.addrs[0].String())
				suite.Require().Equal(resp.Allowances[0].Grantee, suite.addrs[1].String())
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.preRun()
			resp, err := suite.feegrantKeeper.Allowances(suite.ctx, tc.req)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				tc.postRun(resp)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestFeeAllowancesByGranter() {
	testCases := []struct {
		name      string
		req       *feegrant.QueryAllowancesByGranterRequest
		expectErr bool
		preRun    func()
		postRun   func(_ *feegrant.QueryAllowancesByGranterResponse)
	}{
		{
			"nil request",
			nil,
			true,
			func() {},
			func(*feegrant.QueryAllowancesByGranterResponse) {},
		},
		{
			"fail: invalid grantee",
			&feegrant.QueryAllowancesByGranterRequest{
				Granter: invalidGrantee,
			},
			true,
			func() {},
			func(*feegrant.QueryAllowancesByGranterResponse) {},
		},
		{
			"no grants",
			&feegrant.QueryAllowancesByGranterRequest{
				Granter: suite.addrs[0].String(),
			},
			false,
			func() {},
			func(resp *feegrant.QueryAllowancesByGranterResponse) {
				suite.Require().Equal(len(resp.Allowances), 0)
			},
		},
		{
			"valid query: expect single grant",
			&feegrant.QueryAllowancesByGranterRequest{
				Granter: suite.addrs[0].String(),
			},
			false,
			func() {
				suite.grantFeeAllowance(suite.addrs[0], suite.addrs[1])

				// adding this allowance to check whether the pagination working fine.
				suite.grantFeeAllowance(suite.addrs[1], suite.addrs[2])
			},
			func(resp *feegrant.QueryAllowancesByGranterResponse) {
				suite.Require().Equal(len(resp.Allowances), 1)
				suite.Require().Equal(resp.Allowances[0].Granter, suite.addrs[0].String())
				suite.Require().Equal(resp.Allowances[0].Grantee, suite.addrs[1].String())
				suite.Require().Equal(resp.Pagination.Total, uint64(1))
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.preRun()
			resp, err := suite.feegrantKeeper.AllowancesByGranter(suite.ctx, tc.req)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				tc.postRun(resp)
			}
		})
	}
}

func (suite *KeeperTestSuite) grantFeeAllowance(granter, grantee sdk.AccAddress) {
	exp := suite.ctx.BlockTime().AddDate(1, 0, 0)
	err := suite.feegrantKeeper.GrantAllowance(suite.ctx, granter, grantee, &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 555)),
		Expiration: &exp,
	})
	suite.Require().NoError(err)
}

func (suite *KeeperTestSuite) TestDIDAllowance() {
	invalidGranter := "invalid_granter"
	testDID := "did:example:alice"

	testCases := []struct {
		name      string
		req       *feegrant.QueryDIDAllowanceRequest
		expectErr bool
		preRun    func()
		postRun   func(_ *feegrant.QueryDIDAllowanceResponse)
	}{
		{
			"nil request",
			nil,
			true,
			func() {},
			func(*feegrant.QueryDIDAllowanceResponse) {},
		},
		{
			"fail: invalid granter",
			&feegrant.QueryDIDAllowanceRequest{
				Granter:    invalidGranter,
				GranteeDid: testDID,
			},
			true,
			func() {},
			func(*feegrant.QueryDIDAllowanceResponse) {},
		},
		{
			"fail: empty DID",
			&feegrant.QueryDIDAllowanceRequest{
				Granter:    suite.addrs[0].String(),
				GranteeDid: "",
			},
			true,
			func() {},
			func(*feegrant.QueryDIDAllowanceResponse) {},
		},
		{
			"fail: non-existent DID allowance",
			&feegrant.QueryDIDAllowanceRequest{
				Granter:    suite.addrs[0].String(),
				GranteeDid: "did:example:nonexistent",
			},
			true,
			func() {},
			func(*feegrant.QueryDIDAllowanceResponse) {},
		},
		{
			"valid query: single DID grant",
			&feegrant.QueryDIDAllowanceRequest{
				Granter:    suite.addrs[0].String(),
				GranteeDid: testDID,
			},
			false,
			func() {
				suite.grantDIDFeeAllowance(suite.addrs[0], testDID)
			},
			func(resp *feegrant.QueryDIDAllowanceResponse) {
				suite.Require().NotNil(resp.Allowance)
				suite.Require().Equal(resp.Allowance.Granter, suite.addrs[0].String())
				suite.Require().Equal(resp.Allowance.Grantee, testDID)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.preRun()
			resp, err := suite.feegrantKeeper.DIDAllowance(suite.ctx, tc.req)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				tc.postRun(resp)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestDIDAllowancesByGranter() {
	invalidGranter := "invalid_granter"

	testCases := []struct {
		name      string
		req       *feegrant.QueryDIDAllowancesByGranterRequest
		expectErr bool
		preRun    func()
		postRun   func(_ *feegrant.QueryDIDAllowancesByGranterResponse)
	}{
		{
			"nil request",
			nil,
			true,
			func() {},
			func(*feegrant.QueryDIDAllowancesByGranterResponse) {},
		},
		{
			"fail: invalid granter",
			&feegrant.QueryDIDAllowancesByGranterRequest{
				Granter: invalidGranter,
			},
			true,
			func() {},
			func(*feegrant.QueryDIDAllowancesByGranterResponse) {},
		},
		{
			"no DID grants",
			&feegrant.QueryDIDAllowancesByGranterRequest{
				Granter: suite.addrs[0].String(),
			},
			false,
			func() {
				// Grant regular allowance (not DID) to ensure it's not returned
				suite.grantFeeAllowance(suite.addrs[0], suite.addrs[1])
			},
			func(resp *feegrant.QueryDIDAllowancesByGranterResponse) {
				suite.Require().Equal(len(resp.Allowances), 0)
			},
		},
		{
			"valid query: expect DID grants only",
			&feegrant.QueryDIDAllowancesByGranterRequest{
				Granter: suite.addrs[2].String(),
			},
			false,
			func() {
				// Grant regular allowance (should not appear in DID query)
				suite.grantFeeAllowance(suite.addrs[2], suite.addrs[3])

				// Grant DID allowances (should appear in DID query)
				suite.grantDIDFeeAllowance(suite.addrs[2], "did:example:bob")
				suite.grantDIDFeeAllowance(suite.addrs[2], "did:example:alice")
			},
			func(resp *feegrant.QueryDIDAllowancesByGranterResponse) {
				suite.Require().Equal(len(resp.Allowances), 2)
				for _, allowance := range resp.Allowances {
					suite.Require().Equal(allowance.Granter, suite.addrs[2].String())
					suite.Require().True(allowance.Grantee == "did:example:bob" || allowance.Grantee == "did:example:alice")
				}
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.preRun()
			resp, err := suite.feegrantKeeper.DIDAllowancesByGranter(suite.ctx, tc.req)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				tc.postRun(resp)
			}
		})
	}
}

func (suite *KeeperTestSuite) grantDIDFeeAllowance(granter sdk.AccAddress, granteeDID string) {
	exp := suite.ctx.BlockTime().AddDate(1, 0, 0)
	err := suite.feegrantKeeper.GrantDIDAllowance(suite.ctx, granter, granteeDID, &feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uopen", 777)),
		Expiration: &exp,
	})
	suite.Require().NoError(err)
}

func (suite *KeeperTestSuite) TestDIDAllowances() {
	testDID := "did:example:charlie"

	testCases := []struct {
		name      string
		req       *feegrant.QueryDIDAllowancesRequest
		expectErr bool
		preRun    func()
		postRun   func(_ *feegrant.QueryDIDAllowancesResponse)
	}{
		{
			"nil request",
			nil,
			true,
			func() {},
			func(*feegrant.QueryDIDAllowancesResponse) {},
		},
		{
			"fail: empty DID",
			&feegrant.QueryDIDAllowancesRequest{
				GranteeDid: "",
			},
			true,
			func() {},
			func(*feegrant.QueryDIDAllowancesResponse) {},
		},
		{
			"no DID grants for this DID",
			&feegrant.QueryDIDAllowancesRequest{
				GranteeDid: "did:example:nonexistent",
			},
			false,
			func() {},
			func(resp *feegrant.QueryDIDAllowancesResponse) {
				suite.Require().Equal(len(resp.Allowances), 0)
			},
		},
		{
			"valid query: expect single DID grant",
			&feegrant.QueryDIDAllowancesRequest{
				GranteeDid: testDID,
			},
			false,
			func() {
				suite.grantDIDFeeAllowance(suite.addrs[0], testDID)
			},
			func(resp *feegrant.QueryDIDAllowancesResponse) {
				suite.Require().Equal(len(resp.Allowances), 1)
				suite.Require().Equal(resp.Allowances[0].Granter, suite.addrs[0].String())
				suite.Require().Equal(resp.Allowances[0].Grantee, testDID)
			},
		},
		{
			"valid query: expect multiple DID grants from different granters",
			&feegrant.QueryDIDAllowancesRequest{
				GranteeDid: "did:example:multiple",
			},
			false,
			func() {
				// Multiple granters giving allowances to the same DID
				suite.grantDIDFeeAllowance(suite.addrs[0], "did:example:multiple")
				suite.grantDIDFeeAllowance(suite.addrs[1], "did:example:multiple")
				suite.grantDIDFeeAllowance(suite.addrs[2], "did:example:multiple")

				// Grant to a different DID to ensure it's not returned
				suite.grantDIDFeeAllowance(suite.addrs[3], "did:example:other")
			},
			func(resp *feegrant.QueryDIDAllowancesResponse) {
				suite.Require().Equal(len(resp.Allowances), 3)
				granters := make(map[string]bool)
				for _, allowance := range resp.Allowances {
					suite.Require().Equal(allowance.Grantee, "did:example:multiple")
					granters[allowance.Granter] = true
				}
				// Verify all three granters are present
				suite.Require().True(granters[suite.addrs[0].String()])
				suite.Require().True(granters[suite.addrs[1].String()])
				suite.Require().True(granters[suite.addrs[2].String()])
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.preRun()
			resp, err := suite.feegrantKeeper.DIDAllowances(suite.ctx, tc.req)
			if tc.expectErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				tc.postRun(resp)
			}
		})
	}
}
