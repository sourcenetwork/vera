package types

import (
	"fmt"
	time "time"

	"cosmossdk.io/math"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys
var (
	KeyEpochDuration          = []byte("EpochDuration")
	KeyUnlockingEpochs        = []byte("UnlockingEpochs")
	KeyDeveloperPoolFee       = []byte("DeveloperPoolFee")
	KeyInsurancePoolFee       = []byte("InsurancePoolFee")
	KeyInsurancePoolThreshold = []byte("InsurancePoolThreshold")
	KeyProcessRewardsInterval = []byte("ProcessRewardsInterval")
	KeyRewardRates            = []byte("RewardRates")
)

// ParamKeyTable for module parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params object
func NewParams(
	epochDuration *time.Duration,
	unlockingEpochs, developerPoolFee, insurancePoolFee, insurancePoolThreshold, processRewardsInterval int64,
	rewardRates []Rate,
) Params {
	return Params{
		EpochDuration:          epochDuration,
		UnlockingEpochs:        unlockingEpochs,
		DeveloperPoolFee:       developerPoolFee,
		InsurancePoolFee:       insurancePoolFee,
		InsurancePoolThreshold: insurancePoolThreshold,
		ProcessRewardsInterval: processRewardsInterval,
		RewardRates:            rewardRates,
	}
}

// DefaultParams returns default parameters.
// Rate in RewardRates, DeveloperPoolFee, and InsurancePoolFee are integers representing percentages
// with 2 decimal precision (e.g. a rate of 150 represents 150%, a fee of 2 represents 2%).
// InsurancePoolThreshold represents a threshold in "uopen", after which insurance pool is considered "full".
// When the insurance pool is full, the insurance fee is allocated to the developer pool instead.
func DefaultParams() Params {
	epochDuration := DefaultEpochDuration
	return NewParams(
		&epochDuration,                // 5 minutes
		DefaultUnlockingEpochs,        // 2 epochs
		DefaultDeveloperPoolFee,       // 2%
		DefaultInsurancePoolFee,       // 1%
		DefaultInsurancePoolThreshold, // 100,000 open
		DefaultProcessRewardsInterval, // 1000 blocks
		[]Rate{
			{Amount: math.NewInt(300), Rate: 150}, // x1.5 from 300
			{Amount: math.NewInt(200), Rate: 120}, // x1.2 from 200 to 300
			{Amount: math.NewInt(100), Rate: 110}, // x1.1 from 100 to 200
			{Amount: math.NewInt(0), Rate: 100},   // x1.0 from 0 to 100
		},
	)
}

// Validate validates the params
func (p Params) Validate() error {
	if err := validateEpochDuration(p.EpochDuration); err != nil {
		return err
	}
	if err := validateUnlockingEpochs(p.UnlockingEpochs); err != nil {
		return err
	}
	if err := validateDeveloperPoolFee(p.DeveloperPoolFee); err != nil {
		return err
	}
	if err := validateInsurancePoolFee(p.InsurancePoolFee); err != nil {
		return err
	}
	if err := validateInsurancePoolThreshold(p.InsurancePoolThreshold); err != nil {
		return err
	}
	if err := validateProcessRewardsInterval(p.ProcessRewardsInterval); err != nil {
		return err
	}
	if err := validateRewardRates(p.RewardRates); err != nil {
		return err
	}
	return nil
}

// ParamSetPairs implements the ParamSet interface and returns all the key/value pairs
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		{Key: KeyEpochDuration, Value: &p.EpochDuration, ValidatorFn: validateEpochDuration},
		{Key: KeyUnlockingEpochs, Value: &p.UnlockingEpochs, ValidatorFn: validateUnlockingEpochs},
		{Key: KeyDeveloperPoolFee, Value: &p.DeveloperPoolFee, ValidatorFn: validateDeveloperPoolFee},
		{Key: KeyInsurancePoolFee, Value: &p.InsurancePoolFee, ValidatorFn: validateInsurancePoolFee},
		{Key: KeyInsurancePoolThreshold, Value: &p.InsurancePoolThreshold, ValidatorFn: validateInsurancePoolThreshold},
		{Key: KeyProcessRewardsInterval, Value: &p.ProcessRewardsInterval, ValidatorFn: validateProcessRewardsInterval},
		{Key: KeyRewardRates, Value: &p.RewardRates, ValidatorFn: validateRewardRates},
	}
}

func validateEpochDuration(i interface{}) error {
	duration, ok := i.(*time.Duration)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if duration == nil || *duration <= 0 {
		return fmt.Errorf("invalid epoch duration: %d", duration)
	}
	return nil
}

func validateUnlockingEpochs(i interface{}) error {
	epochs, ok := i.(int64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if epochs <= 0 {
		return fmt.Errorf("invalid unlocking epochs: %d", epochs)
	}
	return nil
}

func validateDeveloperPoolFee(i interface{}) error {
	fee, ok := i.(int64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if fee < 0 {
		return fmt.Errorf("invalid developer pool fee: %d", fee)
	}
	return nil
}

func validateInsurancePoolFee(i interface{}) error {
	fee, ok := i.(int64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if fee < 0 {
		return fmt.Errorf("invalid insurance pool fee: %d", fee)
	}
	return nil
}

func validateInsurancePoolThreshold(i interface{}) error {
	threshold, ok := i.(int64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if threshold < 0 {
		return fmt.Errorf("invalid insurance pool threshold: %d", threshold)
	}
	return nil
}

func validateProcessRewardsInterval(i interface{}) error {
	interval, ok := i.(int64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}
	if interval <= 0 {
		return fmt.Errorf("invalid process rewards interval: %d", interval)
	}
	return nil
}

func validateRewardRates(i interface{}) error {
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
