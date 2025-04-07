package metrics

const (
	// global keys
	App     = "sourcehub"
	Count   = "count"
	Error   = "error"
	Latency = "latency"
	Method  = "method"
	Msg     = "msg"
	Status  = "status"
	Tx      = "tx"

	// abci methods
	PrepareProposal = "prepare_proposal"
	ProcessProposal = "process_proposal"

	// acp keys
	Actor = "actor"

	// acp methods
	CheckAccess  = "check_access"
	CreatePolicy = "create_policy"
	EditPolicy   = "edit_policy"

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
)
