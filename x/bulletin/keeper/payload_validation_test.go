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
			payload: []byte(`{"ring_pk":"pk","peer_ids":["peer1","peer2"],"threshold":1,"block_number_nonce":0}`),
		},
		{
			name:    "valid payload with optional fields",
			payload: []byte(`{"ring_pk":"pk","new_peer_ids":["peer3","peer4"],"new_threshold":2,"peer_ids":["peer1","peer2"],"threshold":1,"pss_interval":60,"block_number_nonce":9}`),
		},
		{
			name:    "valid payload with optional null fields",
			payload: []byte(`{"ring_pk":"pk","new_peer_ids":null,"new_threshold":null,"peer_ids":["peer1"],"threshold":1,"pss_interval":null,"block_number_nonce":0}`),
		},
		{
			name:    "unknown fields are accepted",
			payload: []byte(`{"ring_pk":"pk","peer_ids":["peer1"],"threshold":1,"block_number_nonce":0,"extra":"ok"}`),
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
			payload: []byte(`{"ring_pk":"pk","peer_ids":["peer1"],"block_number_nonce":0}`),
			expErr:  true,
		},
		{
			name:    "missing block number nonce",
			payload: []byte(`{"ring_pk":"pk","peer_ids":["peer1"],"threshold":1}`),
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

func TestFinalizeRingPayloadReshare(t *testing.T) {
	currentPayload := []byte(`{"ring_pk":"pk","new_peer_ids":["peer2","peer3"],"new_threshold":2,"peer_ids":["peer1"],"threshold":1,"block_number_nonce":7}`)

	signDocFinalizedPayload, err := deriveFinalizedRingPayloadReshare(currentPayload)
	require.NoError(t, err)
	require.JSONEq(t, `{"ring_pk":"pk","peer_ids":["peer2","peer3"],"threshold":2,"block_number_nonce":7}`, string(signDocFinalizedPayload))

	finalizedPayload, err := finalizeRingPayloadReshare(currentPayload, 42)
	require.NoError(t, err)
	require.JSONEq(t, `{"ring_pk":"pk","peer_ids":["peer2","peer3"],"threshold":2,"block_number_nonce":42}`, string(finalizedPayload))

	currentPayloadWithPSSInterval := []byte(`{"ring_pk":"pk","new_peer_ids":["peer2","peer3"],"new_threshold":2,"peer_ids":["peer1"],"threshold":1,"pss_interval":60,"block_number_nonce":7}`)
	finalizedPayload, err = finalizeRingPayloadReshare(currentPayloadWithPSSInterval, 43)
	require.NoError(t, err)
	require.JSONEq(t, `{"ring_pk":"pk","peer_ids":["peer2","peer3"],"threshold":2,"pss_interval":60,"block_number_nonce":43}`, string(finalizedPayload))

	currentPayloadWithNewPeerIDsOnly := []byte(`{"ring_pk":"pk","new_peer_ids":["peer2","peer3"],"peer_ids":["peer1"],"threshold":1,"block_number_nonce":7}`)
	finalizedPayload, err = finalizeRingPayloadReshare(currentPayloadWithNewPeerIDsOnly, 44)
	require.NoError(t, err)
	require.JSONEq(t, `{"ring_pk":"pk","peer_ids":["peer2","peer3"],"threshold":1,"block_number_nonce":44}`, string(finalizedPayload))

	currentPayloadWithNewThresholdOnly := []byte(`{"ring_pk":"pk","new_threshold":2,"peer_ids":["peer1"],"threshold":1,"block_number_nonce":7}`)
	finalizedPayload, err = finalizeRingPayloadReshare(currentPayloadWithNewThresholdOnly, 45)
	require.NoError(t, err)
	require.JSONEq(t, `{"ring_pk":"pk","peer_ids":["peer1"],"threshold":2,"block_number_nonce":45}`, string(finalizedPayload))

	_, err = finalizeRingPayloadReshare(
		[]byte(`{"ring_pk":"pk","peer_ids":["peer1"],"threshold":1,"block_number_nonce":7}`),
		46,
	)
	require.ErrorIs(t, err, types.ErrInvalidPostPayload)
	require.Contains(t, err.Error(), "missing new_peer_ids or new_threshold")
}
