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
	Amount                 = "amount"
	AutoLockEnabled        = "auto_lock_enabled"
	CreditUtilization      = "credit_utilization"
	CreationHeight         = "creation_height"
	Delegator              = "delegator"
	Developer              = "developer"
	DeveloperPoolBalance   = "developer_pool_balance"
	DstValidator           = "dst_validator"
	Epoch                  = "epoch"
	InsurancePoolBalance   = "insurance_pool_balance"
	TotalCredits           = "total_credits"
	TotalDevGranted        = "total_dev_granted"
	TotalLocked            = "total_locked"
	SrcValidator           = "src_validator"
	SubscriptionAmount     = "subscription_amount"
	SubscriptionExpiration = "subscription_expiration"
	SubscriptionPeriod     = "subscription_period"
	UserDid                = "user_did"
	Validator              = "validator"

	// tier methods
	AddUserSubscription    = "add_user_subscription"
	BurnAllCredits         = "burn_all_credits"
	CancelUnlocking        = "cancel_unlocking"
	CheckDeveloperCredits  = "check_developer_credits"
	CompleteUnlocking      = "complete_unlocking"
	CreateDeveloper        = "create_developer"
	HandleDoubleSign       = "handle_double_sign"
	HandleMissingSignature = "handle_missing_signature"
	Lock                   = "lock"
	ProcessRewards         = "process_rewards"
	Redelegate             = "redelegate"
	RemoveDeveloper        = "remove_developer"
	RemoveUserSubscription = "remove_user_subscription"
	ResetAllCredits        = "reset_all_credits"
	Unlock                 = "unlock"
	UpdateDeveloper        = "update_developer"
	UpdateUserSubscription = "update_user_subscription"

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
