package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	appparams "github.com/sourcenetwork/vera/app/params"
	"github.com/sourcenetwork/vera/testutil/sample"
)

var (
	bondDenom = appparams.DefaultBondDenom
	// bondDenom = appparams.DefaultBondDenom
	delAddr  = sample.RandomAccAddress().String()
	valAddr  = sample.RandomValAddress().String()
	valAddr2 = sample.RandomValAddress().String()
	stake    = sdk.NewCoin(bondDenom, math.NewInt(100))

	invalidAddr       = "invalid_address"
	invalidDenomStake = sdk.NewCoin("denom1", math.NewInt(100))
	nonPositiveStake  = sdk.NewCoin(bondDenom, math.NewInt(0))
	overflowStake     = sdk.NewCoin(bondDenom, math.NewIntFromUint64(1<<63))
)

func TestMsgLock_ValidateBasic(t *testing.T) {

	tests := []struct {
		name string
		msg  MsgLock
		err  error
	}{
		{
			name: "valid request",
			msg: MsgLock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            stake,
			},
		},
		{
			name: "invalid delegator address",
			msg: MsgLock{
				DelegatorAddress: invalidAddr,
				ValidatorAddress: valAddr,
				Stake:            stake,
			},
			err: ErrInvalidAddress,
		}, {
			name: "invalid validator address",
			msg: MsgLock{
				DelegatorAddress: delAddr,
				ValidatorAddress: invalidAddr,
				Stake:            stake,
			},
			err: ErrInvalidAddress,
		}, {
			name: "invalid denom",
			msg: MsgLock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            invalidDenomStake,
			},
			err: ErrInvalidDenom,
		}, {
			name: "invalid denom (non-positive amount)",
			msg: MsgLock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            nonPositiveStake,
			},
			err: ErrInvalidDenom,
		}, {
			name: "invalid denom (overflowed int64 amount)",
			msg: MsgLock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            overflowStake,
			},
			err: ErrInvalidDenom,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgUnlockStake_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgUnlock
		err  error
	}{
		{
			name: "valid request",
			msg: MsgUnlock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            stake,
			},
		},
		{
			name: "invalid delegator address",
			msg: MsgUnlock{
				DelegatorAddress: invalidAddr,
				ValidatorAddress: valAddr,
				Stake:            stake,
			},
			err: ErrInvalidAddress,
		}, {
			name: "invalid validator address",
			msg: MsgUnlock{
				DelegatorAddress: delAddr,
				ValidatorAddress: invalidAddr,
				Stake:            stake,
			},
			err: ErrInvalidAddress,
		}, {
			name: "invalid denom",
			msg: MsgUnlock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            invalidDenomStake,
			},
			err: ErrInvalidDenom,
		}, {
			name: "invalid denom (non-positive amount)",
			msg: MsgUnlock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            nonPositiveStake,
			},
			err: ErrInvalidDenom,
		}, {
			name: "invalid denom (overflowed int64 amount)",
			msg: MsgUnlock{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            overflowStake,
			},
			err: ErrInvalidDenom,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgCancelUnlockingStake_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgCancelUnlocking
		err  error
	}{
		{
			name: "valid request",
			msg: MsgCancelUnlocking{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            stake,
				CreationHeight:   1,
			},
		},
		{
			name: "invalid delegator address",
			msg: MsgCancelUnlocking{
				DelegatorAddress: invalidAddr,
				ValidatorAddress: valAddr,
				Stake:            stake,
				CreationHeight:   1,
			},
			err: ErrInvalidAddress,
		},
		{
			name: "invalid validator address",
			msg: MsgCancelUnlocking{
				DelegatorAddress: delAddr,
				ValidatorAddress: invalidAddr,
				Stake:            stake,
				CreationHeight:   1,
			},
			err: ErrInvalidAddress,
		},
		{
			name: "invalid denom",
			msg: MsgCancelUnlocking{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            invalidDenomStake,
				CreationHeight:   1,
			},
			err: ErrInvalidDenom,
		},
		{
			name: "invalid denom (non-positive amount)",
			msg: MsgCancelUnlocking{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            nonPositiveStake,
				CreationHeight:   1,
			},
			err: ErrInvalidDenom,
		},
		{
			name: "invalid denom (overflowed int64 amount)",
			msg: MsgCancelUnlocking{
				DelegatorAddress: delAddr,
				ValidatorAddress: valAddr,
				Stake:            overflowStake,
				CreationHeight:   1,
			},
			err: ErrInvalidDenom,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMsgRedelegate_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgRedelegate
		err  error
	}{
		{
			name: "valid request",
			msg: MsgRedelegate{
				DelegatorAddress:    delAddr,
				SrcValidatorAddress: valAddr,
				DstValidatorAddress: valAddr2,
				Stake:               stake,
			},
		},
		{
			name: "invalid delegator address",
			msg: MsgRedelegate{
				DelegatorAddress:    invalidAddr,
				SrcValidatorAddress: valAddr,
				DstValidatorAddress: valAddr2,
				Stake:               stake,
			},
			err: ErrInvalidAddress,
		}, {
			name: "invalid src validator address",
			msg: MsgRedelegate{
				DelegatorAddress:    delAddr,
				SrcValidatorAddress: invalidAddr,
				DstValidatorAddress: valAddr2,
				Stake:               stake,
			},
			err: ErrInvalidAddress,
		}, {
			name: "src and dst validator address are the same",
			msg: MsgRedelegate{
				DelegatorAddress:    delAddr,
				SrcValidatorAddress: valAddr,
				DstValidatorAddress: valAddr,
				Stake:               stake,
			},
			err: ErrInvalidAddress,
		}, {
			name: "invalid denom",
			msg: MsgRedelegate{
				DelegatorAddress:    delAddr,
				SrcValidatorAddress: valAddr,
				DstValidatorAddress: valAddr2,
				Stake:               invalidDenomStake,
			},
			err: ErrInvalidDenom,
		}, {
			name: "invalid denom (non-positive amount)",
			msg: MsgRedelegate{
				DelegatorAddress:    delAddr,
				SrcValidatorAddress: valAddr,
				DstValidatorAddress: valAddr2,
				Stake:               nonPositiveStake,
			},
			err: ErrInvalidDenom,
		}, {
			name: "invalid denom (overflowed int64 amount)",
			msg: MsgRedelegate{
				DelegatorAddress:    delAddr,
				SrcValidatorAddress: valAddr,
				DstValidatorAddress: valAddr2,
				Stake:               overflowStake,
			},
			err: ErrInvalidDenom,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
