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

// IgnoreBearerAuthKey stores a flag that indicates whether tx extension option bearer token authority should be ignored.
// The value is parsed from app_state.app_params.ignore_bearer_auth in genesis.json on chain init.
const IgnoreBearerAuthKey = "appparams/ignore_bearer_auth"

// FaucetStoreKey is the store key for faucet data.
const FaucetStoreKey = "faucet"

// AppParamsGenesis defines app-specific params that can be set during genesis.
type AppParamsGenesis struct {
	AllowZeroFeeTxs  bool `json:"allow_zero_fee_txs"`
	IgnoreBearerAuth bool `json:"ignore_bearer_auth"`
}

// FaucetConfig defines the configuration for the faucet service.
type FaucetConfig struct {
	EnableFaucet bool `mapstructure:"enable_faucet"`
}

// Context key for storing extracted DID from JWS extension options.
type contextKey string

const (
	// ExtractedDIDContextKey is the key used to store extracted DID in context
	ExtractedDIDContextKey contextKey = "extracted_did"
)
