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

	DefaultBondDenom   = MicroOpenDenom
	DefaultMinGasPrice = 0.001

	BlocksPerYear       = 31557600
	GoalBonded          = "0.67"
	InflationMin        = "0.02"
	InflationMax        = "0.15"
	InflationRateChange = "0.13"
)
