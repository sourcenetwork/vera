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
	currentRingPayload := requireParsedRingPayload(t, currentPayload)

	signDocFinalizedPayload, err := deriveFinalizedRingPayloadReshare(currentRingPayload)
	require.NoError(t, err)
	require.JSONEq(t, `{"ring_pk":"pk","peer_ids":["peer2","peer3"],"threshold":2,"block_number_nonce":7}`, string(signDocFinalizedPayload))

	finalizedPayload, err := finalizeRingPayloadReshare(currentRingPayload, 42)
	require.NoError(t, err)
	require.JSONEq(t, `{"ring_pk":"pk","peer_ids":["peer2","peer3"],"threshold":2,"block_number_nonce":42}`, string(finalizedPayload))

	currentPayloadWithPSSInterval := []byte(`{"ring_pk":"pk","new_peer_ids":["peer2","peer3"],"new_threshold":2,"peer_ids":["peer1"],"threshold":1,"pss_interval":60,"block_number_nonce":7}`)
	finalizedPayload, err = finalizeRingPayloadReshare(requireParsedRingPayload(t, currentPayloadWithPSSInterval), 43)
	require.NoError(t, err)
	require.JSONEq(t, `{"ring_pk":"pk","peer_ids":["peer2","peer3"],"threshold":2,"pss_interval":60,"block_number_nonce":43}`, string(finalizedPayload))

	currentPayloadWithNewPeerIDsOnly := []byte(`{"ring_pk":"pk","new_peer_ids":["peer2","peer3"],"peer_ids":["peer1"],"threshold":1,"block_number_nonce":7}`)
	finalizedPayload, err = finalizeRingPayloadReshare(requireParsedRingPayload(t, currentPayloadWithNewPeerIDsOnly), 44)
	require.NoError(t, err)
	require.JSONEq(t, `{"ring_pk":"pk","peer_ids":["peer2","peer3"],"threshold":1,"block_number_nonce":44}`, string(finalizedPayload))

	currentPayloadWithNewThresholdOnly := []byte(`{"ring_pk":"pk","new_threshold":2,"peer_ids":["peer1"],"threshold":1,"block_number_nonce":7}`)
	finalizedPayload, err = finalizeRingPayloadReshare(requireParsedRingPayload(t, currentPayloadWithNewThresholdOnly), 45)
	require.NoError(t, err)
	require.JSONEq(t, `{"ring_pk":"pk","peer_ids":["peer1"],"threshold":2,"block_number_nonce":45}`, string(finalizedPayload))

	_, err = finalizeRingPayloadReshare(
		requireParsedRingPayload(t, []byte(`{"ring_pk":"pk","peer_ids":["peer1"],"threshold":1,"block_number_nonce":7}`)),
		46,
	)
	require.ErrorIs(t, err, types.ErrInvalidPostPayload)
	require.Contains(t, err.Error(), "missing new_peer_ids or new_threshold")
}

func TestValidateRingPostUpdate(t *testing.T) {
	ptr := func(v uint32) *uint32 { return &v }

	t.Run("empty inputs are valid", func(t *testing.T) {
		require.NoError(t, validateRingPostUpdate(nil, nil))
		require.NoError(t, validateRingPostUpdate([]string{}, nil))
	})

	t.Run("unique peer ids are valid", func(t *testing.T) {
		require.NoError(t, validateRingPostUpdate([]string{"peer1", "peer2", "peer3"}, nil))
	})

	t.Run("duplicate peer ids are rejected", func(t *testing.T) {
		err := validateRingPostUpdate([]string{"peer1", "peer2", "peer1"}, nil)
		require.ErrorIs(t, err, types.ErrInvalidPostPayload)
		require.Contains(t, err.Error(), "duplicate peer id")
	})

	t.Run("threshold of 1 is valid", func(t *testing.T) {
		require.NoError(t, validateRingPostUpdate(nil, ptr(1)))
	})

	t.Run("threshold above 1 is valid", func(t *testing.T) {
		require.NoError(t, validateRingPostUpdate(nil, ptr(5)))
	})

	t.Run("threshold of 0 is rejected", func(t *testing.T) {
		err := validateRingPostUpdate(nil, ptr(0))
		require.ErrorIs(t, err, types.ErrInvalidPostPayload)
		require.Contains(t, err.Error(), "new_threshold must be at least 1")
	})

	t.Run("both rules enforced together", func(t *testing.T) {
		err := validateRingPostUpdate([]string{"peer1", "peer1"}, ptr(0))
		require.ErrorIs(t, err, types.ErrInvalidPostPayload)
	})
}

func requireParsedRingPayload(t *testing.T, payload []byte) *ringPayloadJSON {
	t.Helper()
	ringPayload, err := parseRingPayloadJSON(payload)
	require.NoError(t, err)
	return ringPayload
}
