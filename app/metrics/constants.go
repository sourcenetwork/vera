package metrics

const (
	// global keys
	App      = "sourcehub"
	Count    = "count"
	Error    = "error"
	Errors   = "errors"
	Internal = "internal"
	Latency  = "latency"
	Method   = "method"
	Msg      = "msg"
	Query    = "query"
	Status   = "status"
	Tx       = "tx"

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
	Amount               = "amount"
	CreditUtilization    = "credit_utilization"
	CreationHeight       = "creation_height"
	Delegator            = "delegator"
	DeveloperPoolBalance = "developer_pool_balance"
	DstValidator         = "dst_validator"
	InsurancePoolBalance = "insurance_pool_balance"
	Epoch                = "epoch"
	TotalLocked          = "total_locked"
	TotalCredits         = "total_credits"
	SrcValidator         = "src_validator"
	Validator            = "validator"

	// tier methods
	BurnAllCredits         = "burn_all_credits"
	CancelUnlocking        = "cancel_unlocking"
	CompleteUnlocking      = "complete_unlocking"
	HandleDoubleSign       = "handle_double_sign"
	HandleMissingSignature = "handle_missing_signature"
	Lock                   = "lock"
	ProcessRewards         = "process_rewards"
	Redelegate             = "redelegate"
	ResetAllCredits        = "reset_all_credits"
	Unlock                 = "unlock"

	// ChainIDEnvVar represents the environment variable, which when set,
	// is used as the chain id value for metric collection
	ChainIDEnvVar = "CHAIN_ID"
)

var (
	SourcehubMethodSeconds     []string = []string{App, Method, SecondsUnit}
	SourcehubMethodTotal       []string = []string{App, Method, CounterSuffix}
	SourcehubMethodErrorsTotal []string = []string{App, Method, Errors, CounterSuffix}

	SourcehubMsgSeconds     []string = []string{App, Msg, SecondsUnit}
	SourcehubMsgTotal       []string = []string{App, Msg, CounterSuffix}
	SourcehubMsgErrorsTotal []string = []string{App, Msg, Errors, CounterSuffix}

	SourcehubQuerySeconds     []string = []string{App, Query, SecondsUnit}
	SourcehubQueryTotal       []string = []string{App, Query, CounterSuffix}
	SourcehubQueryErrorsTotal []string = []string{App, Query, Errors, CounterSuffix}

	SourcehubInternalErrorsTotal []string = []string{App, Errors, Internal, CounterSuffix}
)
