package metrics

const (
	Count       = "count"
	Epoch       = "epoch"
	Latency     = "latency"
	Method      = "method"
	Transaction = "transaction"

	PrepareProposal = "prepare_proposal"
	ProcessProposal = "process_proposal"

	BurnAllCredits    = "burn_all_credits"
	CancelUnlocking   = "cancel_unlocking"
	CompleteUnlocking = "complete_unlocking"
	Lock              = "lock"
	Redelegate        = "redelegate"
	ResetAllCredits   = "reset_all_credits"
	Unlock            = "unlock"

	CreditUtilization = "credit_utilization"
	TotalLocked       = "total_locked"
	TotalCredits      = "total_credits"

	CheckAccess  = "check_access"
	CreatePolicy = "create_policy"
	EditPolicy   = "edit_policy"
)
