package metrics

const (
	Count       = "count"
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

	TotalCredits = "total_credits"
	TotalLocked  = "total_locked"

	CheckAccess  = "check_access"
	CreatePolicy = "create_policy"
	EditPolicy   = "edit_policy"
)
