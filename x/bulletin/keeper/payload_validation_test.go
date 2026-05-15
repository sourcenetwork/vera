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
	empty := &ringPayloadJSON{}

	withExistingNewPeers := func(peers []string) *ringPayloadJSON {
		return &ringPayloadJSON{NewPeerIDs: &peers}
	}
	withExistingNewThreshold := func(t uint32) *ringPayloadJSON {
		return &ringPayloadJSON{NewThreshold: ptr(t)}
	}
	withExistingBoth := func(peers []string, thresh uint32) *ringPayloadJSON {
		return &ringPayloadJSON{NewPeerIDs: &peers, NewThreshold: ptr(thresh)}
	}

	t.Run("empty inputs are valid", func(t *testing.T) {
		require.NoError(t, validateRingPostUpdate(nil, nil, empty))
		require.NoError(t, validateRingPostUpdate([]string{}, nil, empty))
	})

	t.Run("unique peer ids are valid", func(t *testing.T) {
		require.NoError(t, validateRingPostUpdate([]string{"peer1", "peer2", "peer3"}, nil, empty))
	})

	t.Run("duplicate peer ids are rejected", func(t *testing.T) {
		err := validateRingPostUpdate([]string{"peer1", "peer2", "peer1"}, nil, empty)
		require.ErrorIs(t, err, types.ErrInvalidPostPayload)
		require.Contains(t, err.Error(), "duplicate peer id")
	})

	t.Run("threshold of 1 is valid", func(t *testing.T) {
		require.NoError(t, validateRingPostUpdate(nil, ptr(1), empty))
	})

	t.Run("threshold above 1 is valid", func(t *testing.T) {
		require.NoError(t, validateRingPostUpdate(nil, ptr(5), empty))
	})

	t.Run("threshold of 0 is rejected", func(t *testing.T) {
		err := validateRingPostUpdate(nil, ptr(0), empty)
		require.ErrorIs(t, err, types.ErrInvalidPostPayload)
		require.Contains(t, err.Error(), "new_threshold must be at least 1")
	})

	t.Run("new_peer_ids count >= new_threshold in same message is valid", func(t *testing.T) {
		require.NoError(t, validateRingPostUpdate([]string{"p1", "p2", "p3"}, ptr(3), empty))
	})

	t.Run("new_peer_ids count < new_threshold in same message is rejected", func(t *testing.T) {
		err := validateRingPostUpdate([]string{"p1", "p2"}, ptr(3), empty)
		require.ErrorIs(t, err, types.ErrInvalidPostPayload)
		require.Contains(t, err.Error(), "new_peer_ids count")
	})

	t.Run("reshare in progress: new_peer_ids blocked when payload has new_threshold", func(t *testing.T) {
		err := validateRingPostUpdate([]string{"p1"}, nil, withExistingNewThreshold(3))
		require.ErrorIs(t, err, types.ErrReshareInProgress)
	})

	t.Run("reshare in progress: new_threshold blocked when payload has new_peer_ids", func(t *testing.T) {
		err := validateRingPostUpdate(nil, ptr(5), withExistingNewPeers([]string{"p1", "p2"}))
		require.ErrorIs(t, err, types.ErrReshareInProgress)
	})

	t.Run("reshare in progress: new_peer_ids blocked even if count satisfies threshold", func(t *testing.T) {
		err := validateRingPostUpdate([]string{"p1", "p2", "p3"}, nil, withExistingNewThreshold(2))
		require.ErrorIs(t, err, types.ErrReshareInProgress)
	})

	t.Run("no check when neither side has threshold set", func(t *testing.T) {
		require.NoError(t, validateRingPostUpdate([]string{"p1"}, nil, empty))
	})

	t.Run("no check when neither side has peer ids set", func(t *testing.T) {
		require.NoError(t, validateRingPostUpdate(nil, ptr(3), empty))
	})

	t.Run("existing both fields consistent passes through unchanged", func(t *testing.T) {
		require.NoError(t, validateRingPostUpdate(nil, nil, withExistingBoth([]string{"p1", "p2"}, 2)))
	})
}

func requireParsedRingPayload(t *testing.T, payload []byte) *ringPayloadJSON {
	t.Helper()
	ringPayload, err := parseRingPayloadJSON(payload)
	require.NoError(t, err)
	return ringPayload
}
