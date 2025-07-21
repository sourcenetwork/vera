package params

const (
	MicroOpenDenom      = "uopen"
	MicroOpenDenomAlias = "microopen"
	OpenDenom           = "open"
	OpenName            = "Source Open"
	OpenSymbol          = "OPEN"
	OpenDescription     = "OPEN is the native staking token of SourceHub."

	MicroCreditDenom      = "ucredit"
	MicroCreditDenomAlias = "microcredit"
	CreditDenom           = "credit"
	CreditName            = "Source Credit"
	CreditSymbol          = "CREDIT"
	CreditDescription     = "Credit is the utility token for access services on SourceHub. Non-transferable."
	CreditFeeMultiplier   = 10

	DefaultBondDenom   = MicroOpenDenom
	DefaultMinGasPrice = "0.001"

	BlocksPerYear       = 31557600
	GoalBonded          = "0.67"
	InflationMin        = "0.02"
	InflationMax        = "0.15"
	InflationRateChange = "0.13"
	InitialInflation    = "0.13"

	// AppParamsGenesisKey is the Genesis' file "app_state" key name to set SourceHub app_params
	AppParamsGenesisKey = "app_params"
)

// AllowZeroFeeTxsKey stores a flag that indicates whether zero-fee transactions are allowed.
// The value is parsed from app_state.app_params.allow_zero_fee_txs in genesis.json on chain init.
const AllowZeroFeeTxsKey = "appparams/allow_zero_fee_txs"

// EnableFaucetKey stores a flag that indicates whether the faucet is enabled.
// The value is parsed from app_state.app_params.enable_faucet in genesis.json on chain init.
const EnableFaucetKey = "appparams/enable_faucet"

// FaucetStoreKey is the store key for faucet data.
const FaucetStoreKey = "faucet"

// AppParamsGenesis defines app-specific params that can be set during genesis.
type AppParamsGenesis struct {
	AllowZeroFeeTxs bool `json:"allow_zero_fee_txs"`
	EnableFaucet    bool `json:"enable_faucet"`
}
