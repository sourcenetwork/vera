package types

import (
	"testing"

	"github.com/sourcenetwork/immutable"
	"github.com/stretchr/testify/require"
)

func TestGenerateDocumentID_CrossImplVectorAndMalleability(t *testing.T) {
	const d1 = `{"enc_cmt":[1,2,3],"encrypted_data":[4,5,6],"nonce":[0,0,0,0,0,0,0,0,0,0,0,0]}`
	const d2 = "{ \"nonce\" : [0,0,0,0,0,0,0,0,0,0,0,0],\n  \"enc_cmt\": [1, 2, 3] ,\"encrypted_data\":[4,5,6] , \"extra\":\"ignored\" }"
	const p = `{"challenge":[7,8],"response":[9,10]}`

	id1, err := GenerateDocumentID("ring-1", d1, p, "policy-b", "document", "read", immutable.Some("gold"), immutable.Some[uint64](1700000000))
	require.NoError(t, err)
	id2, err := GenerateDocumentID("ring-1", d2, p, "policy-b", "document", "read", immutable.Some("gold"), immutable.Some[uint64](1700000000))
	require.NoError(t, err)

	require.Equal(t, id1, id2, "non-canonical JSON must not change the id")
	require.Equal(t, "a5f065b5d5e02043d5427455daa3809b59d85990b60e7b5fa11dbcbbe2692fbe", id1)

	_, err = GenerateDocumentID("ring-1", "{}", p, "p", "r", "x", immutable.None[string](), immutable.None[uint64]())
	require.Error(t, err, "missing document fields must be rejected")
	_, err = GenerateDocumentID("ring-1", d1, "not json", "p", "r", "x", immutable.None[string](), immutable.None[uint64]())
	require.Error(t, err, "malformed proof must be rejected")
}
