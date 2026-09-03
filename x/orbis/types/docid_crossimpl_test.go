package types

import (
	"testing"

	"github.com/sourcenetwork/immutable"
	"github.com/stretchr/testify/require"
)

// Must stay byte-identical with orbis-rs `generate_document_id`: whitespace and
// field order do not change the id, and any extra / differently-cased key is
// rejected on both sides.
func TestGenerateDocumentID_CrossImplVectorAndMalleability(t *testing.T) {
	const d1 = `{"enc_cmt":[1,2,3],"encrypted_data":[4,5,6],"nonce":[0,0,0,0,0,0,0,0,0,0,0,0]}`
	const d2 = "{ \"nonce\" : [0,0,0,0,0,0,0,0,0,0,0,0],\n  \"enc_cmt\": [1, 2, 3] ,\"encrypted_data\":[4,5,6] }"
	const p = `{"challenge":[7,8],"response":[9,10]}`

	id := func(doc, proof string) (string, error) {
		return GenerateDocumentID("ring-1", doc, proof, "policy-b", "document", "read",
			immutable.Some("gold"), immutable.Some[uint64](1700000000))
	}

	id1, err := id(d1, p)
	require.NoError(t, err)
	id2, err := id(d2, p)
	require.NoError(t, err)
	require.Equal(t, id1, id2, "whitespace/order must not change the id")
	require.Equal(t, "a5f065b5d5e02043d5427455daa3809b59d85990b60e7b5fa11dbcbbe2692fbe", id1)

	// Rejections — each must fail identically on the Rust side.
	_, err = id("{}", p)
	require.Error(t, err, "missing document fields")
	_, err = id(d1, "not json")
	require.Error(t, err, "malformed proof")
	_, err = id(`{"enc_cmt":[1,2,3],"encrypted_data":[4,5,6],"nonce":[0,0,0,0,0,0,0,0,0,0,0,0],"extra":1}`, p)
	require.Error(t, err, "unknown field")
	_, err = id(`{"enc_cmt":[1,2,3],"ENC_CMT":[9,9,9],"encrypted_data":[4,5,6],"nonce":[0,0,0,0,0,0,0,0,0,0,0,0]}`, p)
	require.Error(t, err, "case-variant duplicate key must not be foldable into a second id")
	_, err = id(d1, `{"challenge":[7,8],"response":[9,10],"Response":[0]}`)
	require.Error(t, err, "case-variant duplicate key in proof")
}
