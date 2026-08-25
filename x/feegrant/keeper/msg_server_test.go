package keeper_test

import (
	"fmt"
	"time"

	"github.com/golang/mock/gomock"

	"github.com/sourcenetwork/vera/x/feegrant"

	codecaddress "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (suite *KeeperTestSuite) TestGrantAllowance() {
	ctx := suite.ctx.WithBlockTime(time.Now())
	oneYear := ctx.BlockTime().AddDate(1, 0, 0)
	yesterday := ctx.BlockTime().AddDate(0, 0, -1)

	addressCodec := codecaddress.NewBech32Codec("cosmos")

	testCases := []struct {
		name      string
		req       func() *feegrant.MsgGrantAllowance
		expectErr bool
		errMsg    string
	}{
		{
			"invalid granter address",
			func() *feegrant.MsgGrantAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{})
				suite.Require().NoError(err)
				invalid := "invalid-granter"
				return &feegrant.MsgGrantAllowance{
					Granter:   invalid,
					Grantee:   suite.addrs[1].String(),
					Allowance: any,
				}
			},
			true,
			"decoding bech32 failed",
		},
		{
			"invalid grantee address",
			func() *feegrant.MsgGrantAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{})
				suite.Require().NoError(err)
				invalid := "invalid-grantee"
				return &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[0].String(),
					Grantee:   invalid,
					Allowance: any,
				}
			},
			true,
			"decoding bech32 failed",
		},
		{
			"valid: grantee account doesn't exist",
			func() *feegrant.MsgGrantAllowance {
				grantee := "cosmos139f7kncmglres2nf3h4hc4tade85ekfr8sulz5"
				granteeAccAddr, err := addressCodec.StringToBytes(grantee)
				suite.Require().NoError(err)
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.coins,
					Expiration: &oneYear,
				})
				suite.Require().NoError(err)

				suite.accountKeeper.EXPECT().GetAccount(gomock.Any(), granteeAccAddr).Return(nil).AnyTimes()

				acc := authtypes.NewBaseAccountWithAddress(granteeAccAddr)
				add, err := addressCodec.StringToBytes(grantee)
				suite.Require().NoError(err)

				suite.accountKeeper.EXPECT().NewAccountWithAddress(gomock.Any(), add).Return(acc).AnyTimes()
				suite.accountKeeper.EXPECT().SetAccount(gomock.Any(), acc).Return()

				suite.Require().NoError(err)
				return &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[0].String(),
					Grantee:   grantee,
					Allowance: any,
				}
			},
			false,
			"",
		},
		{
			"invalid: past expiry",
			func() *feegrant.MsgGrantAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.coins,
					Expiration: &yesterday,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[0].String(),
					Grantee:   suite.addrs[1].String(),
					Allowance: any,
				}
			},
			true,
			"expiration is before current block time",
		},
		{
			"valid: basic fee allowance",
			func() *feegrant.MsgGrantAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.coins,
					Expiration: &oneYear,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[0].String(),
					Grantee:   suite.addrs[1].String(),
					Allowance: any,
				}
			},
			false,
			"",
		},
		{
			"fail: fee allowance exists",
			func() *feegrant.MsgGrantAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.coins,
					Expiration: &oneYear,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[0].String(),
					Grantee:   suite.addrs[1].String(),
					Allowance: any,
				}
			},
			true,
			"fee allowance already exists",
		},
		{
			"valid: periodic fee allowance",
			func() *feegrant.MsgGrantAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.PeriodicAllowance{
					Basic: feegrant.BasicAllowance{
						SpendLimit: suite.coins,
						Expiration: &oneYear,
					},
					PeriodSpendLimit: suite.coins,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[1].String(),
					Grantee:   suite.addrs[2].String(),
					Allowance: any,
				}
			},
			false,
			"",
		},
		{
			"error: fee allowance exists",
			func() *feegrant.MsgGrantAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.PeriodicAllowance{
					Basic: feegrant.BasicAllowance{
						SpendLimit: suite.coins,
						Expiration: &oneYear,
					},
					PeriodSpendLimit: suite.coins,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[1].String(),
					Grantee:   suite.addrs[2].String(),
					Allowance: any,
				}
			},
			true,
			"fee allowance already exists",
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			_, err := suite.msgSrvr.GrantAllowance(ctx, tc.req())
			if tc.expectErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errMsg)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestRevokeAllowance() {
	oneYear := suite.ctx.BlockTime().AddDate(1, 0, 0)

	testCases := []struct {
		name      string
		request   *feegrant.MsgRevokeAllowance
		preRun    func()
		expectErr bool
		errMsg    string
	}{
		{
			"error: invalid granter",
			&feegrant.MsgRevokeAllowance{
				Granter: invalidGranter,
				Grantee: suite.addrs[1].String(),
			},
			func() {},
			true,
			"decoding bech32 failed",
		},
		{
			"error: invalid grantee",
			&feegrant.MsgRevokeAllowance{
				Granter: suite.addrs[0].String(),
				Grantee: invalidGrantee,
			},
			func() {},
			true,
			"decoding bech32 failed",
		},
		{
			"error: fee allowance not found",
			&feegrant.MsgRevokeAllowance{
				Granter: suite.addrs[0].String(),
				Grantee: suite.addrs[1].String(),
			},
			func() {},
			true,
			"fee-grant not found",
		},
		{
			"success: revoke fee allowance",
			&feegrant.MsgRevokeAllowance{
				Granter: suite.addrs[0].String(),
				Grantee: suite.addrs[1].String(),
			},
			func() {
				// removing fee allowance from previous tests if exists
				suite.msgSrvr.RevokeAllowance(suite.ctx, &feegrant.MsgRevokeAllowance{
					Granter: suite.addrs[0].String(),
					Grantee: suite.addrs[1].String(),
				})

				any, err := codectypes.NewAnyWithValue(&feegrant.PeriodicAllowance{
					Basic: feegrant.BasicAllowance{
						SpendLimit: suite.coins,
						Expiration: &oneYear,
					},
					PeriodSpendLimit: suite.coins,
				})
				suite.Require().NoError(err)
				req := &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[0].String(),
					Grantee:   suite.addrs[1].String(),
					Allowance: any,
				}
				_, err = suite.msgSrvr.GrantAllowance(suite.ctx, req)
				suite.Require().NoError(err)
			},
			false,
			"",
		},
		{
			"error: check fee allowance revoked",
			&feegrant.MsgRevokeAllowance{
				Granter: suite.addrs[0].String(),
				Grantee: suite.addrs[1].String(),
			},
			func() {},
			true,
			"fee-grant not found",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.preRun()
			_, err := suite.msgSrvr.RevokeAllowance(suite.ctx, tc.request)
			if tc.expectErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errMsg)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestPruneAllowances() {
	ctx := suite.ctx.WithBlockTime(time.Now())
	oneYear := ctx.BlockTime().AddDate(1, 0, 0)

	// We create 76 allowances, all expiring in one year
	count := 0
	for i := 0; i < len(suite.addrs); i++ {
		for j := 0; j < len(suite.addrs); j++ {
			if count == 76 {
				break
			}
			if suite.addrs[i].String() == suite.addrs[j].String() {
				continue
			}

			any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
				SpendLimit: suite.coins,
				Expiration: &oneYear,
			})
			suite.Require().NoError(err)
			req := &feegrant.MsgGrantAllowance{
				Granter:   suite.addrs[i].String(),
				Grantee:   suite.addrs[j].String(),
				Allowance: any,
			}

			_, err = suite.msgSrvr.GrantAllowance(ctx, req)
			if err != nil {
				// do not fail, just try with another pair
				continue
			}

			count++
		}
	}

	// we have 76 allowances
	count = 0
	err := suite.feegrantKeeper.IterateAllFeeAllowances(ctx, func(grant feegrant.Grant) bool {
		count++
		return false
	})
	suite.Require().NoError(err)
	suite.Require().Equal(76, count)

	// after a year and one day passes, they are all expired
	oneYearAndADay := ctx.BlockTime().AddDate(1, 0, 1)
	ctx = suite.ctx.WithBlockTime(oneYearAndADay)

	// we prune them, but currently only 75 will be pruned
	_, err = suite.msgSrvr.PruneAllowances(ctx, &feegrant.MsgPruneAllowances{})
	suite.Require().NoError(err)

	// we have 1 allowance left
	count = 0
	err = suite.feegrantKeeper.IterateAllFeeAllowances(ctx, func(grant feegrant.Grant) bool {
		count++
		return false
	})
	suite.Require().NoError(err)
	suite.Require().Equal(1, count)
}

func (suite *KeeperTestSuite) TestGrantDIDAllowance() {
	ctx := suite.ctx.WithBlockTime(time.Now())
	oneYear := ctx.BlockTime().AddDate(1, 0, 0)

	testCases := []struct {
		name      string
		req       func() *feegrant.MsgGrantDIDAllowance
		expectErr bool
		errMsg    string
		preRun    func()
	}{
		{
			"valid DID allowance grant",
			func() *feegrant.MsgGrantDIDAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.coins,
					Expiration: &oneYear,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantDIDAllowance{
					Granter:    suite.addrs[0].String(),
					GranteeDid: "did:example:bob",
					Allowance:  any,
				}
			},
			false,
			"",
			func() {},
		},
		{
			"invalid granter address",
			func() *feegrant.MsgGrantDIDAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.coins,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantDIDAllowance{
					Granter:    "invalid",
					GranteeDid: "did:example:alice",
					Allowance:  any,
				}
			},
			true,
			"decoding bech32 failed",
			func() {},
		},
		{
			"empty DID",
			func() *feegrant.MsgGrantDIDAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.coins,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantDIDAllowance{
					Granter:    suite.addrs[0].String(),
					GranteeDid: "",
					Allowance:  any,
				}
			},
			true,
			"invalid DID",
			func() {},
		},
		{
			"duplicate DID allowance",
			func() *feegrant.MsgGrantDIDAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.coins,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantDIDAllowance{
					Granter:    suite.addrs[1].String(),
					GranteeDid: "did:example:bob",
					Allowance:  any,
				}
			},
			true,
			"DID allowance already exists",
			func() {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.coins,
				})
				suite.Require().NoError(err)
				req := &feegrant.MsgGrantDIDAllowance{
					Granter:    suite.addrs[1].String(),
					GranteeDid: "did:example:bob",
					Allowance:  any,
				}
				_, err = suite.msgSrvr.GrantDIDAllowance(ctx, req)
				suite.Require().NoError(err)
			},
		},
		{
			"invalid DID - no colon",
			func() *feegrant.MsgGrantDIDAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.coins,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantDIDAllowance{
					Granter:    suite.addrs[0].String(),
					GranteeDid: "invalid-did-format",
					Allowance:  any,
				}
			},
			true,
			"invalid DID",
			func() {},
		},
		{
			"invalid DID - wrong prefix",
			func() *feegrant.MsgGrantDIDAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.coins,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantDIDAllowance{
					Granter:    suite.addrs[0].String(),
					GranteeDid: "notdid:example:alice",
					Allowance:  any,
				}
			},
			true,
			"invalid DID",
			func() {},
		},
		{
			"DID allowance with past expiration",
			func() *feegrant.MsgGrantDIDAllowance {
				yesterday := ctx.BlockTime().AddDate(0, 0, -1)
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.coins,
					Expiration: &yesterday,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantDIDAllowance{
					Granter:    suite.addrs[0].String(),
					GranteeDid: "did:example:future",
					Allowance:  any,
				}
			},
			true,
			"expiration is before current block time",
			func() {},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.preRun()
			req := tc.req()

			// Call ValidateBasic first
			if err := req.ValidateBasic(); err != nil {
				if tc.expectErr {
					suite.Require().Error(err)
					suite.Require().Contains(err.Error(), tc.errMsg)
					return
				} else {
					suite.Require().NoError(err)
				}
			}

			_, err := suite.msgSrvr.GrantDIDAllowance(ctx, req)
			if tc.expectErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errMsg)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestExpireDIDAllowance() {
	ctx := suite.ctx.WithBlockTime(time.Now())
	oneYear := ctx.BlockTime().AddDate(1, 0, 0)

	testCases := []struct {
		name      string
		req       func() *feegrant.MsgExpireDIDAllowance
		preRun    func()
		expectErr bool
		errMsg    string
	}{
		{
			"invalid granter address",
			func() *feegrant.MsgExpireDIDAllowance {
				return &feegrant.MsgExpireDIDAllowance{
					Granter:    "invalid",
					GranteeDid: "did:example:alice",
				}
			},
			func() {},
			true,
			"decoding bech32 failed",
		},
		{
			"empty DID",
			func() *feegrant.MsgExpireDIDAllowance {
				return &feegrant.MsgExpireDIDAllowance{
					Granter:    suite.addrs[0].String(),
					GranteeDid: "",
				}
			},
			func() {},
			true,
			"invalid DID",
		},
		{
			"DID allowance not found",
			func() *feegrant.MsgExpireDIDAllowance {
				return &feegrant.MsgExpireDIDAllowance{
					Granter:    suite.addrs[0].String(),
					GranteeDid: "did:example:bob",
				}
			},
			func() {},
			true,
			"not found",
		},
		{
			"success: expire DID allowance",
			func() *feegrant.MsgExpireDIDAllowance {
				return &feegrant.MsgExpireDIDAllowance{
					Granter:    suite.addrs[2].String(),
					GranteeDid: "did:example:bob",
				}
			},
			func() {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.coins,
					Expiration: &oneYear,
				})
				suite.Require().NoError(err)
				req := &feegrant.MsgGrantDIDAllowance{
					Granter:    suite.addrs[2].String(),
					GranteeDid: "did:example:bob",
					Allowance:  any,
				}
				_, err = suite.msgSrvr.GrantDIDAllowance(ctx, req)
				suite.Require().NoError(err)
			},
			false,
			"",
		},
		{
			"error: check DID allowance expired",
			func() *feegrant.MsgExpireDIDAllowance {
				return &feegrant.MsgExpireDIDAllowance{
					Granter:    suite.addrs[2].String(),
					GranteeDid: "did:example:bob",
				}
			},
			func() {},
			false,
			"",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.preRun()
			req := tc.req()

			// Call ValidateBasic first
			if err := req.ValidateBasic(); err != nil {
				if tc.expectErr {
					suite.Require().Error(err)
					suite.Require().Contains(err.Error(), tc.errMsg)
					return
				}
				suite.Require().NoError(err)
			}

			_, err := suite.msgSrvr.ExpireDIDAllowance(ctx, req)
			if tc.expectErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errMsg)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}
func (suite *KeeperTestSuite) TestPruneDIDAllowances() {
	ctx := suite.ctx.WithBlockTime(time.Now())
	oneYear := ctx.BlockTime().AddDate(1, 0, 0)

	// We create 100 DID allowances, all expiring in one year
	count := 0
	for i := 0; i < len(suite.addrs) && count < 100; i++ {
		did := fmt.Sprintf("did:example:test%d", i)

		any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
			SpendLimit: suite.coins,
			Expiration: &oneYear,
		})
		suite.Require().NoError(err)
		req := &feegrant.MsgGrantDIDAllowance{
			Granter:    suite.addrs[i].String(),
			GranteeDid: did,
			Allowance:  any,
		}

		_, err = suite.msgSrvr.GrantDIDAllowance(ctx, req)
		if err != nil {
			// do not fail, just try with another address
			continue
		}

		count++
	}

	// we have some DID allowances
	countBefore := 0
	err := suite.feegrantKeeper.IterateAllDIDAllowances(ctx, func(grant feegrant.DIDGrant) bool {
		countBefore++
		return false
	})
	suite.Require().NoError(err)
	suite.Require().True(countBefore >= count)

	// after a year and one day passes, they are all expired
	oneYearAndADay := ctx.BlockTime().AddDate(1, 0, 1)
	ctx = suite.ctx.WithBlockTime(oneYearAndADay)

	// we prune them, currently up to 75 will be pruned (same as regular allowances)
	_, err = suite.msgSrvr.PruneDIDAllowances(ctx, &feegrant.MsgPruneDIDAllowances{
		Pruner: suite.addrs[0].String(),
	})
	suite.Require().NoError(err)

	// count remaining DID allowances after pruning
	countAfter := 0
	err = suite.feegrantKeeper.IterateAllDIDAllowances(ctx, func(grant feegrant.DIDGrant) bool {
		countAfter++
		return false
	})
	suite.Require().NoError(err)

	// Verify that some allowances were pruned (should be fewer than before)
	suite.Require().True(countAfter < countBefore, "some DID allowances should have been pruned")
}
