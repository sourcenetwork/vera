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
	CreditDescription     = "Credit is the utility token for access services on SourceHub. Non transferrable."
	CreditFeeMultiplier   = 10

	DefaultBondDenom   = MicroOpenDenom
	DefaultMinGasPrice = "0.001"

	BlocksPerYear       = 31557600
	GoalBonded          = "0.67"
	InflationMin        = "0.02"
	InflationMax        = "0.15"
	InflationRateChange = "0.13"
	InitialInflation    = "0.13"
)

// AllowZeroFeeTxsKey stores a flag that indicates whether zero-fee transactions are allowed.
// The value is parsed from app_state.app_params.allow_zero_fee_txs in genesis.json on chain init.
const AllowZeroFeeTxsKey = "appparams/allow_zero_fee_txs"

// AppParamsGenesis defines app-specific params that can be set during genesis.
type AppParamsGenesis struct {
	AllowZeroFeeTxs bool `json:"allow_zero_fee_txs"`
}
