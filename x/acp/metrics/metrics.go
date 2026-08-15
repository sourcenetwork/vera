package metrics

const (

	// counter
	// label for grpc method
	// not sure - host? weird because scraping multiple targets would skew the distribution since everyone executes this msg
	MsgTotal = "vera_acp_msg_total"

	// counter
	// label for grpc method
	MsgErrors = "vera_acp_msg_errors_total"

	// histogram
	// label for grpc method
	MsgSeconds = "vera_acp_msg_seconds"

	// counter
	InvariantViolation = "vera_acp_invariant_violation_total"

	// counter
	// label for grpc method
	QueryTotal = "vera_acp_query_total"

	// counter
	// label for grpc method (?)
	QueryErrors = "vera_acp_query_errors_total"

	// histogram
	// label for grpc method
	// label for error or not
	QuerySeconds = "vera_acp_query_seconds"
)
