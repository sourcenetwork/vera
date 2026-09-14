package keeper

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/sourcenetwork/vera/x/orbis/types"
)

const acceptedReportSessionTestChainID = "vera-accepted-report-session-test"

// seedAcceptedReportSession marks the session that req would dedupe against as
// accepted, computing the same key AcceptedReportSession itself derives (see
// reportSessionDedupeID), so the query can be exercised against a real
// accepted-session record without going through full report submission.
func seedAcceptedReportSession(t *testing.T, ctx sdk.Context, k Keeper, req *types.QueryAcceptedReportSessionRequest) {
	t.Helper()
	sessionID, err := reportSessionDedupeID(
		&types.ReportEnvelope{
			ChainId:        ctx.ChainID(),
			RingId:         req.RingId,
			ReportType:     req.ReportType,
			AccusedNodeKey: req.AccusedNodeKey,
			SessionId:      req.SessionId,
		},
		reportPayload{
			originProtocol: req.OriginProtocol,
			attemptID:      req.AttemptId,
		},
	)
	require.NoError(t, err)
	k.SetAcceptedReportPair(ctx, "report-"+sessionID, sessionID, 1_000_000)
}

func TestKeeper_AcceptedReportSession(t *testing.T) {
	validReq := func() *types.QueryAcceptedReportSessionRequest {
		return &types.QueryAcceptedReportSessionRequest{
			RingId:         "ring1",
			ReportType:     "node_offline",
			OriginProtocol: "pre",
			AccusedNodeKey: "accused1",
			SessionId:      "session1",
		}
	}

	tests := []struct {
		name         string
		req          *types.QueryAcceptedReportSessionRequest
		seed         bool
		wantCode     codes.Code
		wantErr      string
		wantAccepted bool
	}{
		{
			name:     "nil request",
			req:      nil,
			wantCode: codes.InvalidArgument,
			wantErr:  "invalid request",
		},
		{
			name: "missing ring id",
			req: func() *types.QueryAcceptedReportSessionRequest {
				r := validReq()
				r.RingId = ""
				return r
			}(),
			wantCode: codes.InvalidArgument,
			wantErr:  types.ErrInvalidRingId.Error(),
		},
		{
			name: "missing report type",
			req: func() *types.QueryAcceptedReportSessionRequest {
				r := validReq()
				r.ReportType = ""
				return r
			}(),
			wantCode: codes.InvalidArgument,
			wantErr:  "report_type is required",
		},
		{
			name: "missing origin protocol",
			req: func() *types.QueryAcceptedReportSessionRequest {
				r := validReq()
				r.OriginProtocol = ""
				return r
			}(),
			wantCode: codes.InvalidArgument,
			wantErr:  "origin_protocol is required",
		},
		{
			name: "missing accused node key",
			req: func() *types.QueryAcceptedReportSessionRequest {
				r := validReq()
				r.AccusedNodeKey = ""
				return r
			}(),
			wantCode: codes.InvalidArgument,
			wantErr:  "accused_node_key is required",
		},
		{
			name: "missing session id",
			req: func() *types.QueryAcceptedReportSessionRequest {
				r := validReq()
				r.SessionId = ""
				return r
			}(),
			wantCode: codes.InvalidArgument,
			wantErr:  "session_id is required",
		},
		{
			name:         "unaccepted session, empty attempt id",
			req:          validReq(),
			wantAccepted: false,
		},
		{
			name:         "accepted session, empty attempt id",
			req:          validReq(),
			seed:         true,
			wantAccepted: true,
		},
		{
			name: "unaccepted session, non-empty attempt id",
			req: func() *types.QueryAcceptedReportSessionRequest {
				r := validReq()
				r.SessionId = "session-attempt"
				r.AttemptId = bytes.Repeat([]byte{7}, 32)
				return r
			}(),
			wantAccepted: false,
		},
		{
			name: "accepted session, non-empty attempt id",
			req: func() *types.QueryAcceptedReportSessionRequest {
				r := validReq()
				r.SessionId = "session-attempt-accepted"
				r.AttemptId = bytes.Repeat([]byte{7}, 32)
				return r
			}(),
			seed:         true,
			wantAccepted: true,
		},
	}

	k, _, ctx := setupOrbisKeeper(t)
	ctx = ctx.WithChainID(acceptedReportSessionTestChainID)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.seed {
				seedAcceptedReportSession(t, ctx, k, tt.req)
			}

			resp, err := k.AcceptedReportSession(ctx, tt.req)

			if tt.wantErr != "" {
				require.Error(t, err)
				require.Equal(t, tt.wantCode, status.Code(err))
				require.Contains(t, err.Error(), tt.wantErr)
				require.Nil(t, resp)
				return
			}

			require.NoError(t, err)
			require.Equal(t, &types.QueryAcceptedReportSessionResponse{Accepted: tt.wantAccepted}, resp)
		})
	}
}
