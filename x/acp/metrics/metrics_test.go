package metrics

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricConstants(t *testing.T) {
	require.Equal(t, "vera_acp_msg_total", MsgTotal)
	require.Equal(t, "vera_acp_msg_errors_total", MsgErrors)
	require.Equal(t, "vera_acp_msg_seconds", MsgSeconds)
	require.Equal(t, "vera_acp_invariant_violation_total", InvariantViolation)
	require.Equal(t, "vera_acp_query_total", QueryTotal)
	require.Equal(t, "vera_acp_query_errors_total", QueryErrors)
	require.Equal(t, "vera_acp_query_seconds", QuerySeconds)
}
