package metrics

const (
	Count       = "count"
	Latency     = "latency"
	Method      = "method"
	Transaction = "transaction"

	PrepareProposal = "prepare_proposal"
	ProcessProposal = "process_proposal"

	CancelUnlocking   = "cancel_unlocking"
	CompleteUnlocking = "complete_unlocking"
	Lock              = "lock"
	Redelegate        = "redelegate"
	Unlock            = "unlock"

	TotalLocked = "total_locked"

	CheckAccess  = "check_access"
	CreatePolicy = "create_policy"
	EditPolicy   = "edit_policy"
)
