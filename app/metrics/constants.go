package metrics

const (
	// global keys
	App     = "sourcehub"
	Count   = "count"
	Error   = "error"
	Latency = "latency"
	Method  = "method"
	Msg     = "msg"
	Query   = "query"
	Status  = "status"
	Tx      = "tx"

	// Units
	SecondsUnit   = "seconds"
	CounterSuffix = "total"

	// Labels
	HostnameLabel = "host"
	ChainIDLabel  = "chain_id"
	ModuleLabel   = "module"
	EndpointLabel = "endpoint"

	// abci methods
	PrepareProposal = "prepare_proposal"
	ProcessProposal = "process_proposal"

	// tier keys
	Amount            = "amount"
	CreditUtilization = "credit_utilization"
	CreationHeight    = "creation_height"
	Delegator         = "delegator"
	DstValidator      = "dst_validator"
	Epoch             = "epoch"
	TotalLocked       = "total_locked"
	TotalCredits      = "total_credits"
	SrcValidator      = "src_validator"
	Validator         = "validator"

	// tier methods
	BurnAllCredits    = "burn_all_credits"
	CancelUnlocking   = "cancel_unlocking"
	CompleteUnlocking = "complete_unlocking"
	Lock              = "lock"
	Redelegate        = "redelegate"
	ResetAllCredits   = "reset_all_credits"
	Unlock            = "unlock"

	// ChainIDEnvVar represents the environment variable, which when set,
	// is used as the chain id value for metric collection
	ChainIDEnvVar = "CHAIN_ID"
)

var (
	SourcehubMsgSeconds     []string = []string{"sourcehub", Msg, SecondsUnit}
	SourcehubMsgTotal       []string = []string{"sourcehub", Msg, CounterSuffix}
	SourcehubMsgErrorsTotal []string = []string{"sourcehub", Msg, "errors", CounterSuffix}

	SourcehubQuerySeconds     []string = []string{"sourcehub", Query, SecondsUnit}
	SourcehubQueryTotal       []string = []string{"sourcehub", Query, CounterSuffix}
	SourcehubQueryErrorsTotal []string = []string{"sourcehub", Query, "errors", CounterSuffix}
)
