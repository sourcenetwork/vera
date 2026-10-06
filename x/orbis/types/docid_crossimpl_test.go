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
			immutable.Some("gold"), immutable.Some[uint64](1700000000),
			immutable.None[string](), immutable.None[string]())
	}

	id1, err := id(d1, p)
	require.NoError(t, err)
	id2, err := id(d2, p)
	require.NoError(t, err)
	require.Equal(t, id1, id2, "whitespace/order must not change the id")
	require.Equal(t, "e555cfcb145edf3d4cd8acbae93e05dc3a48eb0162b3af90f42064ab837c9a06", id1)

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
	_, err = id(`{"enc_cmt":[1,2,3],"enc_cmt":[9,9,9],"encrypted_data":[4,5,6],"nonce":[0,0,0,0,0,0,0,0,0,0,0,0]}`, p)
	require.Error(t, err, "exact duplicate key must not collapse silently (Serde errors 'duplicate field')")
	_, err = id(`{"enc_cmt":null,"encrypted_data":[4,5,6],"nonce":[0,0,0,0,0,0,0,0,0,0,0,0]}`, p)
	require.Error(t, err, "explicit null value (Serde rejects null for Vec<u8>)")
	_, err = id(`{"enc_cmt":[1,null,3],"encrypted_data":[4,5,6],"nonce":[0,0,0,0,0,0,0,0,0,0,0,0]}`, p)
	require.Error(t, err, "null array element (Serde rejects null for u8)")
	_, err = id(d1, p+"  trailing")
	require.Error(t, err, "trailing data after the JSON object")
}

// The PET tag attachment must be present or absent *together*, must change
// the id when present, and — critically — must leave an ordinary (non-PET)
// document's id byte-identical to before the attachment existed, since Rust
// and every existing document computed their id without it. Must stay
// byte-identical with orbis-rs's `generate_document_id_pet_attachment_presence_and_binding`.
func TestGenerateDocumentID_PetAttachmentPresenceAndBinding(t *testing.T) {
	const d1 = `{"enc_cmt":[1,2,3],"encrypted_data":[4,5,6],"nonce":[0,0,0,0,0,0,0,0,0,0,0,0]}`
	const p = `{"challenge":[7,8],"response":[9,10]}`
	const tag = `{"ephemeral_point":[1,1],"masked_fingerprint":[2,2]}`
	const tagProof = `{"challenge":[3,3],"response":[4,4]}`

	id := func(petTag, petTagProof immutable.Option[string]) (string, error) {
		return GenerateDocumentID("ring-1", d1, p, "policy-b", "document", "read",
			immutable.Some("gold"), immutable.Some[uint64](1700000000),
			petTag, petTagProof)
	}

	withoutPet, err := id(immutable.None[string](), immutable.None[string]())
	require.NoError(t, err)
	// Must reproduce the exact id from the vector above — a document with no
	// PET attachment must hash identically to one that never had the field.
	require.Equal(t, "e555cfcb145edf3d4cd8acbae93e05dc3a48eb0162b3af90f42064ab837c9a06", withoutPet)

	withPet, err := id(immutable.Some(tag), immutable.Some(tagProof))
	require.NoError(t, err)
	require.NotEqual(t, withoutPet, withPet, "a present PET attachment must change the id")

	// Presence must be all-or-nothing.
	_, err = id(immutable.Some(tag), immutable.None[string]())
	require.Error(t, err, "tag without its proof")
	_, err = id(immutable.None[string](), immutable.Some(tagProof))
	require.Error(t, err, "proof without its tag")

	// Malformed PET JSON must be rejected the same way document/proof are.
	_, err = id(immutable.Some("not json"), immutable.Some(tagProof))
	require.Error(t, err)
	_, err = id(immutable.Some(tag), immutable.Some("not json"))
	require.Error(t, err)
	_, err = id(immutable.Some(`{"ephemeral_point":[1,1],"masked_fingerprint":[2,2],"extra":1}`), immutable.Some(tagProof))
	require.Error(t, err, "unknown field in pet_tag")

	// Every field of the attachment participates in the binding.
	otherTag, err := id(immutable.Some(`{"ephemeral_point":[9,9],"masked_fingerprint":[2,2]}`), immutable.Some(tagProof))
	require.NoError(t, err)
	require.NotEqual(t, withPet, otherTag, "changing ephemeral_point must change the id")
	otherProof, err := id(immutable.Some(tag), immutable.Some(`{"challenge":[9,9],"response":[4,4]}`))
	require.NoError(t, err)
	require.NotEqual(t, withPet, otherProof, "changing the tag knowledge proof must change the id")
}
