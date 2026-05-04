package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

func TestValidateRingPayloadJSON(t *testing.T) {
	testCases := []struct {
		name    string
		payload []byte
		expErr  bool
	}{
		{
			name:    "minimal valid payload",
			payload: []byte(`{"ring_pk":"pk","peer_ids":["peer1","peer2"],"threshold":1}`),
		},
		{
			name:    "valid payload with optional fields",
			payload: []byte(`{"ring_pk":"pk","next_peer_ids":["peer3","peer4"],"new_threshold":2,"peer_ids":["peer1","peer2"],"threshold":1,"pss_interval":60}`),
		},
		{
			name:    "valid payload with optional null fields",
			payload: []byte(`{"ring_pk":"pk","next_peer_ids":null,"new_threshold":null,"peer_ids":["peer1"],"threshold":1,"pss_interval":null}`),
		},
		{
			name:    "unknown fields are accepted",
			payload: []byte(`{"ring_pk":"pk","peer_ids":["peer1"],"threshold":1,"extra":"ok"}`),
		},
		{
			name:    "invalid json",
			payload: []byte(`not-json`),
			expErr:  true,
		},
		{
			name:    "missing ring_pk",
			payload: []byte(`{"peer_ids":["peer1"],"threshold":1}`),
			expErr:  true,
		},
		{
			name:    "missing peer_ids",
			payload: []byte(`{"ring_pk":"pk","threshold":1}`),
			expErr:  true,
		},
		{
			name:    "missing threshold",
			payload: []byte(`{"ring_pk":"pk","peer_ids":["peer1"]}`),
			expErr:  true,
		},
		{
			name:    "required null field is rejected",
			payload: []byte(`{"ring_pk":"pk","peer_ids":null,"threshold":1}`),
			expErr:  true,
		},
		{
			name:    "negative threshold is rejected",
			payload: []byte(`{"ring_pk":"pk","peer_ids":["peer1"],"threshold":-1}`),
			expErr:  true,
		},
		{
			name:    "non-string peer id is rejected",
			payload: []byte(`{"ring_pk":"pk","peer_ids":["peer1",2],"threshold":1}`),
			expErr:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRingPayloadJSON(tc.payload)
			if tc.expErr {
				require.ErrorIs(t, err, types.ErrInvalidPostPayload)
				return
			}
			require.NoError(t, err)
		})
	}
}
