package test

import (
	"context"
	"reflect"
	"time"

	"cosmossdk.io/math"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/golang/mock/gomock"
	epochstypes "github.com/sourcenetwork/sourcehub/x/epochs/types"
)

// Mock bank keeper
type MockBankKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockBankKeeperMockRecorder
}

type MockBankKeeperMockRecorder struct {
	mock *MockBankKeeper
}

func NewMockBankKeeper(ctrl *gomock.Controller) *MockBankKeeper {
	mock := &MockBankKeeper{ctrl: ctrl}
	mock.recorder = &MockBankKeeperMockRecorder{mock}
	return mock
}

func (m *MockBankKeeper) EXPECT() *MockBankKeeperMockRecorder {
	return m.recorder
}

func (m *MockBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetBalance", ctx, addr, denom)
	ret0 := ret[0].(sdk.Coin)
	return ret0
}

func (mr *MockBankKeeperMockRecorder) GetBalance(ctx, addr, denom interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetBalance", reflect.TypeOf((*MockBankKeeper)(nil).GetBalance), ctx, addr, denom)
}

func (m *MockBankKeeper) SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SendCoins", ctx, fromAddr, toAddr, amt)
	return nil
}

func (mr *MockBankKeeperMockRecorder) SendCoins(ctx, fromAddr, toAddr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoins", reflect.TypeOf((*MockBankKeeper)(nil).SendCoins), ctx, fromAddr, toAddr, amt)
}

func (m *MockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "BurnCoins", ctx, moduleName, amt)
	return nil
}

func (mr *MockBankKeeperMockRecorder) BurnCoins(ctx, moduleName, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BurnCoins", reflect.TypeOf((*MockBankKeeper)(nil).BurnCoins), ctx, moduleName, amt)
}

func (m *MockBankKeeper) MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MintCoins", ctx, moduleName, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockBankKeeperMockRecorder) MintCoins(ctx, moduleName, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MintCoins", reflect.TypeOf((*MockBankKeeper)(nil).MintCoins), ctx, moduleName, amt)
}

func (m *MockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendCoinsFromModuleToAccount", ctx, senderModule, recipientAddr, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockBankKeeperMockRecorder) SendCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoinsFromModuleToAccount", reflect.TypeOf((*MockBankKeeper)(nil).SendCoinsFromModuleToAccount), ctx, senderModule, recipientAddr, amt)
}

func (m *MockBankKeeper) DelegateCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DelegateCoinsFromAccountToModule", ctx, senderAddr, recipientModule, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockBankKeeperMockRecorder) DelegateCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DelegateCoinsFromAccountToModule", reflect.TypeOf((*MockBankKeeper)(nil).DelegateCoinsFromAccountToModule), ctx, senderAddr, recipientModule, amt)
}

func (m *MockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendCoinsFromAccountToModule", ctx, senderAddr, recipientModule, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockBankKeeperMockRecorder) SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendCoinsFromAccountToModule", reflect.TypeOf((*MockBankKeeper)(nil).SendCoinsFromAccountToModule), ctx, senderAddr, recipientModule, amt)
}

func (m *MockBankKeeper) UndelegateCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UndelegateCoinsFromModuleToAccount", ctx, senderModule, recipientAddr, amt)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockBankKeeperMockRecorder) UndelegateCoinsFromModuleToAccount(ctx, senderModule, recipientAddr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UndelegateCoinsFromModuleToAccount", reflect.TypeOf((*MockBankKeeper)(nil).UndelegateCoinsFromModuleToAccount), ctx, senderModule, recipientAddr, amt)
}

func (m *MockBankKeeper) IterateAllBalances(ctx context.Context, cb func(addr sdk.AccAddress, coin sdk.Coin) bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "IterateAllBalances", ctx, cb)
}

func (mr *MockBankKeeperMockRecorder) IterateAllBalances(ctx, cb interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IterateAllBalances", reflect.TypeOf((*MockBankKeeper)(nil).IterateAllBalances), ctx, cb)
}

// Mock distribution keeper
type MockDistributionKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockDistributionKeeperRecorder
}

type MockDistributionKeeperRecorder struct {
	mock *MockDistributionKeeper
}

func NewMockDistributionKeeper(ctrl *gomock.Controller) *MockDistributionKeeper {
	mock := &MockDistributionKeeper{ctrl: ctrl}
	mock.recorder = &MockDistributionKeeperRecorder{mock}
	return mock
}

func (m *MockDistributionKeeper) EXPECT() *MockDistributionKeeperRecorder {
	return m.recorder
}

func (m *MockDistributionKeeper) GetPreviousProposerConsAddr(ctx context.Context) (sdk.ConsAddress, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetPreviousProposerConsAddr", ctx)
	ret0 := ret[0].(sdk.ConsAddress)
	return ret0, nil
}

func (mr *MockDistributionKeeperRecorder) GetPreviousProposerConsAddr(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetPreviousProposerConsAddr", reflect.TypeOf((*MockDistributionKeeper)(nil).GetPreviousProposerConsAddr), ctx)
}

func (m *MockDistributionKeeper) AllocateTokensToValidator(ctx context.Context, val stakingtypes.ValidatorI, tokens sdk.DecCoins) error {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AllocateTokensToValidator", ctx, val, tokens)
	return nil
}

func (m *MockDistributionKeeper) AllocateTokens(ctx context.Context, totalReward int64, bondedValidators []abcitypes.VoteInfo) error {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AllocateTokensToValidator", ctx, totalReward, bondedValidators)
	return nil
}

func (m *MockDistributionKeeper) GetValidatorOutstandingRewards(ctx context.Context, valAddr sdk.ValAddress) (distrtypes.ValidatorOutstandingRewards, error) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "GetValidatorOutstandingRewards", ctx, valAddr)
	return distrtypes.ValidatorOutstandingRewards{}, nil
}

// Mock staking keeper
type MockStakingKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockStakingKeeperRecorder
}

type MockStakingKeeperRecorder struct {
	mock *MockStakingKeeper
}

func NewMockStakingKeeper(ctrl *gomock.Controller) *MockStakingKeeper {
	mock := &MockStakingKeeper{ctrl: ctrl}
	mock.recorder = &MockStakingKeeperRecorder{mock}
	return mock
}

func (m *MockStakingKeeper) EXPECT() *MockStakingKeeperRecorder {
	return m.recorder
}

func (m *MockStakingKeeper) GetValidatorByConsAddr(ctx context.Context, consAddr sdk.ConsAddress) (stakingtypes.Validator, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetValidatorByConsAddr", ctx, consAddr)
	ret0 := ret[0].(stakingtypes.Validator)
	return ret0, nil
}

func (mr *MockStakingKeeperRecorder) GetValidatorByConsAddr(ctx, consAddr any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValidatorByConsAddr", reflect.TypeOf((*MockStakingKeeper)(nil).GetValidatorByConsAddr), ctx, consAddr)
}

func (m *MockStakingKeeper) GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetValidator", ctx, addr)
	ret0, _ := ret[0].(stakingtypes.Validator)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStakingKeeperRecorder) GetValidator(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetValidator", reflect.TypeOf((*MockStakingKeeper)(nil).GetValidator), ctx, addr)
}

func (m *MockStakingKeeper) IterateValidators(ctx context.Context, fn func(index int64, validator stakingtypes.ValidatorI) (stop bool)) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IterateValidators", ctx, fn)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockStakingKeeperRecorder) IterateValidators(ctx, fn interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IterateValidators", reflect.TypeOf((*MockStakingKeeper)(nil).IterateValidators), ctx, fn)
}

func (m *MockStakingKeeper) TotalBondedTokens(ctx context.Context) (math.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TotalBondedTokens", ctx)
	ret0, _ := ret[0].(math.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStakingKeeperRecorder) TotalBondedTokens(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TotalBondedTokens", reflect.TypeOf((*MockStakingKeeper)(nil).TotalBondedTokens), ctx)
}

func (m *MockStakingKeeper) Delegate(ctx context.Context, delAddr sdk.AccAddress, bondAmt math.Int, tokenSrc stakingtypes.BondStatus, validator stakingtypes.Validator, subtractAccount bool) (math.LegacyDec, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delegate", ctx, delAddr, bondAmt, tokenSrc, validator, subtractAccount)
	ret0, _ := ret[0].(math.LegacyDec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStakingKeeperRecorder) Delegate(ctx, delAddr, bondAmt, tokenSrc, validator, subtractAccount interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delegate", reflect.TypeOf((*MockStakingKeeper)(nil).Delegate), ctx, delAddr, bondAmt, tokenSrc, validator, subtractAccount)
}

func (m *MockStakingKeeper) BeginRedelegation(ctx context.Context, delAddr sdk.AccAddress, valSrcAddr, valDstAddr sdk.ValAddress,
	sharesAmount math.LegacyDec) (completionTime time.Time, err error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BeginRedelegation", ctx, delAddr, valSrcAddr, valDstAddr, sharesAmount)
	ret0, _ := ret[0].(time.Time)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStakingKeeperRecorder) BeginRedelegation(ctx, delAddr, valSrcAddr, valDstAddr, sharesAmount interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BeginRedelegation", reflect.TypeOf((*MockStakingKeeper)(nil).BeginRedelegation), ctx, delAddr, valSrcAddr, valDstAddr, sharesAmount)
}

func (m *MockStakingKeeper) BondDenom(ctx context.Context) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "BondDenom", ctx)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStakingKeeperRecorder) BondDenom(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "BondDenom", reflect.TypeOf((*MockStakingKeeper)(nil).BondDenom), ctx)
}

func (m *MockStakingKeeper) CompleteUnbonding(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (sdk.Coins, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CompleteUnbonding", ctx, delAddr, valAddr)
	ret0, _ := ret[0].(sdk.Coins)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStakingKeeperRecorder) CompleteUnbonding(ctx, delAddr, valAddr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CompleteUnbonding", reflect.TypeOf((*MockStakingKeeper)(nil).CompleteUnbonding), ctx, delAddr, valAddr)
}

func (m *MockStakingKeeper) GetUnbondingDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (
	ubd stakingtypes.UnbondingDelegation, err error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUnbondingDelegation", ctx, delAddr, valAddr)
	ret0, _ := ret[0].(stakingtypes.UnbondingDelegation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStakingKeeperRecorder) GetUnbondingDelegation(ctx, delAddr, valAddr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUnbondingDelegation", reflect.TypeOf((*MockStakingKeeper)(nil).GetUnbondingDelegation), ctx, delAddr, valAddr)
}

func (m *MockStakingKeeper) RemoveUnbondingDelegation(ctx context.Context, ubd stakingtypes.UnbondingDelegation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveUnbondingDelegation", ctx, ubd)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockStakingKeeperRecorder) RemoveUnbondingDelegation(ctx, ubd interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveUnbondingDelegation", reflect.TypeOf((*MockStakingKeeper)(nil).RemoveUnbondingDelegation), ctx, ubd)
}

func (m *MockStakingKeeper) SetUnbondingDelegation(ctx context.Context, ubd stakingtypes.UnbondingDelegation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetUnbondingDelegation", ctx, ubd)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockStakingKeeperRecorder) SetUnbondingDelegation(ctx, ubd interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetUnbondingDelegation", reflect.TypeOf((*MockStakingKeeper)(nil).SetUnbondingDelegation), ctx, ubd)
}

func (m *MockStakingKeeper) SetValidator(ctx context.Context, addr stakingtypes.Validator) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetValidator", ctx, addr)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockStakingKeeperRecorder) SetValidator(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetValidator", reflect.TypeOf((*MockStakingKeeper)(nil).SetValidator), ctx, addr)
}

func (m *MockStakingKeeper) SetValidatorByConsAddr(ctx context.Context, addr stakingtypes.Validator) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetValidatorByConsAddr", ctx, addr)
	ret0, _ := ret[0].(error)
	return ret0
}

func (mr *MockStakingKeeperRecorder) SetValidatorByConsAddr(ctx, addr interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetValidatorByConsAddr", reflect.TypeOf((*MockStakingKeeper)(nil).SetValidatorByConsAddr), ctx, addr)
}

func (m *MockStakingKeeper) Undelegate(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, sharesAmount math.LegacyDec) (
	time.Time, math.Int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Undelegate", ctx, delAddr, valAddr, sharesAmount)
	ret0, _ := ret[0].(time.Time)
	ret1, _ := ret[1].(math.Int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

func (mr *MockStakingKeeperRecorder) Undelegate(ctx, delAddr, valAddr, sharesAmount interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Undelegate", reflect.TypeOf((*MockStakingKeeper)(nil).Undelegate), ctx, delAddr, valAddr, sharesAmount)
}

func (m *MockStakingKeeper) ValidateUnbondAmount(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int) (
	shares math.LegacyDec, err error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ValidateUnbondAmount", ctx, delAddr, valAddr, amt)
	ret0, _ := ret[0].(math.LegacyDec)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

func (mr *MockStakingKeeperRecorder) ValidateUnbondAmount(ctx, delAddr, valAddr, amt interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ValidateUnbondAmount", reflect.TypeOf((*MockStakingKeeper)(nil).ValidateUnbondAmount), ctx, delAddr, valAddr, amt)
}

// Mock epochs keeper
type MockEpochsKeeper struct {
	ctrl     *gomock.Controller
	recorder *MockEpochsKeeperMockRecorder
}

type MockEpochsKeeperMockRecorder struct {
	mock *MockEpochsKeeper
}

func NewMockEpochsKeeper(ctrl *gomock.Controller) *MockEpochsKeeper {
	mock := &MockEpochsKeeper{ctrl: ctrl}
	mock.recorder = &MockEpochsKeeperMockRecorder{mock}
	return mock
}

func (m *MockEpochsKeeper) EXPECT() *MockEpochsKeeperMockRecorder {
	return m.recorder
}

func (m *MockEpochsKeeper) GetEpochInfo(ctx context.Context, identifier string) epochstypes.EpochInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetEpochInfo", ctx, identifier)
	ret0, _ := ret[0].(epochstypes.EpochInfo)
	return ret0
}

func (mr *MockEpochsKeeperMockRecorder) GetEpochInfo(ctx, identifier interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetEpochInfo", reflect.TypeOf((*MockEpochsKeeper)(nil).GetEpochInfo), ctx, identifier)
}

func (m *MockEpochsKeeper) SetEpochInfo(ctx context.Context, info epochstypes.EpochInfo) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetEpochInfo", ctx, info)
}

func (mr *MockEpochsKeeperMockRecorder) SetEpochInfo(ctx, info interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(
		mr.mock, "SetEpochInfo",
		reflect.TypeOf((*MockEpochsKeeper)(nil).SetEpochInfo),
		ctx, info,
	)
}
