package types

import (
	"fmt"
	time "time"

	"cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys
var (
	KeyEpochDuration    = []byte("EpochDuration")
	KeyUnlockingEpochs  = []byte("UnlockingEpochs")
	KeyCreditRewardRate = []byte("RewardRates")
)

// ParamKeyTable for module parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params object
func NewParams(epochDuration *time.Duration, unlockingEpochs int64, creditRewardRate []Rate) Params {
	return Params{
		EpochDuration:   epochDuration,
		UnlockingEpochs: unlockingEpochs,
		RewardRates:     creditRewardRate,
	}
}

// DefaultParams returns default parameters.
// Rates in RewardRates are integers representing percentages with 2 decimal precision.
// For example, a rate of 150 represents 1.50 (150 / 100).
func DefaultParams() Params {
	du := 5 * time.Minute
	return Params{
		EpochDuration:   &du,
		UnlockingEpochs: 2,
		RewardRates: []Rate{
			{Amount: math.NewInt(300), Rate: 150},
			{Amount: math.NewInt(200), Rate: 120},
			{Amount: math.NewInt(100), Rate: 110},
			{Amount: math.NewInt(0), Rate: 100},
		},
	}
}

// Validate validates the params
func (p Params) Validate() error {
	if err := validateEpochDuration(p.EpochDuration); err != nil {
		return err
	}
	if err := validateCreditRewardRate(p.RewardRates); err != nil {
		return err
	}
	return nil
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		{Key: KeyEpochDuration, Value: &p.EpochDuration, ValidatorFn: validateEpochDuration},
		{Key: KeyUnlockingEpochs, Value: &p.UnlockingEpochs, ValidatorFn: validateUnlockingEpochs},
		{Key: KeyCreditRewardRate, Value: &p.RewardRates, ValidatorFn: validateCreditRewardRate},
	}
}

func validateEpochDuration(i interface{}) error {
	_, ok := i.(*time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateUnlockingEpochs(i interface{}) error {
	_, ok := i.(int64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	return nil
}

func validateCreditRewardRate(i interface{}) error {
	rates, ok := i.([]Rate)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	for _, rate := range rates {
		if rate.Amount.IsNegative() {
			return fmt.Errorf("invalid locked stake: %s", rate.Amount)
		}
		if rate.Rate <= 0 {
			return fmt.Errorf("invalid rate: %d", rate.Rate)
		}
	}
	return nil
}
