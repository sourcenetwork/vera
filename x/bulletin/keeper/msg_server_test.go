package keeper

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	blst "github.com/supranational/blst/bindings/go"

	"github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

func TestMsgUpdateRingPostByAcp_ValidateBasic(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	ownerKey := secp256k1.GenPrivKey().PubKey()
	owner := authtypes.NewBaseAccount(sdk.AccAddress(ownerKey.Address()), ownerKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, owner)

	cases := []struct {
		msg    *types.MsgUpdateRingPostByAcp
		errMsg string
	}{
		{&types.MsgUpdateRingPostByAcp{}, "invalid creator address"},
		{&types.MsgUpdateRingPostByAcp{Creator: owner.Address}, "invalid namespace id"},
		{&types.MsgUpdateRingPostByAcp{Creator: owner.Address, Namespace: "ns1"}, "invalid post id"},
	}
	for _, c := range cases {
		err := c.msg.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), c.errMsg)
	}
}

func TestMsgUpdateRingPostByAcp(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	ownerKey := secp256k1.GenPrivKey().PubKey()
	owner := authtypes.NewBaseAccount(sdk.AccAddress(ownerKey.Address()), ownerKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, owner)

	outsiderKey := secp256k1.GenPrivKey().PubKey()
	outsider := authtypes.NewBaseAccount(sdk.AccAddress(outsiderKey.Address()), outsiderKey, 2, 1)
	k.accountKeeper.SetAccount(ctx, outsider)

	collabKey := secp256k1.GenPrivKey().PubKey()
	collab := authtypes.NewBaseAccount(sdk.AccAddress(collabKey.Address()), collabKey, 3, 1)
	k.accountKeeper.SetAccount(ctx, collab)

	setupTestPolicy(t, ctx, k)

	namespace := "ns1"
	namespaceId := getNamespaceId(namespace)

	_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   owner.Address,
		Namespace: namespace,
	})
	require.NoError(t, err)

	policyId := k.GetPolicyId(ctx)
	require.NotEmpty(t, policyId)

	ringPayloadBytes := makeRingPayloadWithPolicy(t, policyId)
	_, err = k.CreatePost(ctx, &types.MsgCreatePost{
		Creator:   owner.Address,
		Namespace: namespace,
		Payload:   ringPayloadBytes,
	})
	require.NoError(t, err)

	postId := types.GeneratePostId(namespaceId, ringPayloadBytes)

	t.Run("namespace not found", func(t *testing.T) {
		_, err := k.UpdateRingPostByAcp(ctx, &types.MsgUpdateRingPostByAcp{
			Creator:    owner.Address,
			Namespace:  "does-not-exist",
			PostId:     postId,
			NewPeerIds: []string{"peer2"},
		})
		require.ErrorIs(t, err, types.ErrNamespaceNotFound)
	})

	t.Run("post not found", func(t *testing.T) {
		_, err := k.UpdateRingPostByAcp(ctx, &types.MsgUpdateRingPostByAcp{
			Creator:    owner.Address,
			Namespace:  namespace,
			PostId:     "nonexistent-post-id",
			NewPeerIds: []string{"peer2"},
		})
		require.ErrorIs(t, err, types.ErrPostNotFound)
	})

	t.Run("non-ring payload is rejected", func(t *testing.T) {
		invalidPostID := "invalid-ring-payload-post"
		k.SetPost(ctx, types.Post{
			Id:        invalidPostID,
			Namespace: namespaceId,
			Payload:   []byte("not-ring-json"),
		})
		_, err := k.UpdateRingPostByAcp(ctx, &types.MsgUpdateRingPostByAcp{
			Creator:    owner.Address,
			Namespace:  namespace,
			PostId:     invalidPostID,
			NewPeerIds: []string{"peer2"},
		})
		require.ErrorIs(t, err, types.ErrInvalidPostPayload)
	})

	t.Run("ring payload without policy_id is rejected", func(t *testing.T) {
		noPolicyPostID := "no-policy-id-post"
		noPolicyPayload := ringPayloadWithSeed(t, "sourcehub-bulletin-bls-test-seed-0003", []string{"peer1"}, 1, nil, nil)
		k.SetPost(ctx, types.Post{
			Id:        noPolicyPostID,
			Namespace: namespaceId,
			Payload:   noPolicyPayload,
		})
		_, err := k.UpdateRingPostByAcp(ctx, &types.MsgUpdateRingPostByAcp{
			Creator:    owner.Address,
			Namespace:  namespace,
			PostId:     noPolicyPostID,
			NewPeerIds: []string{"peer2"},
		})
		require.ErrorIs(t, err, types.ErrRingPayloadMissingPolicyId)
	})

	t.Run("outsider without permission is rejected", func(t *testing.T) {
		_, err := k.UpdateRingPostByAcp(ctx, &types.MsgUpdateRingPostByAcp{
			Creator:    outsider.Address,
			Namespace:  namespace,
			PostId:     postId,
			NewPeerIds: []string{"peer2"},
		})
		require.ErrorIs(t, err, types.ErrInvalidPostUpdater)

		stored := k.getPost(ctx, namespaceId, postId)
		require.NotNil(t, stored)
		require.Equal(t, ringPayloadBytes, stored.Payload)
	})

	t.Run("collaborator can update new_peer_ids and post_id is preserved", func(t *testing.T) {
		_, err := k.AddCollaborator(ctx, &types.MsgAddCollaborator{
			Creator:      owner.Address,
			Namespace:    namespace,
			Collaborator: collab.Address,
		})
		require.NoError(t, err)

		t.Run("duplicate new_peer_ids are rejected", func(t *testing.T) {
			_, err := k.UpdateRingPostByAcp(ctx, &types.MsgUpdateRingPostByAcp{
				Creator:    collab.Address,
				Namespace:  namespace,
				PostId:     postId,
				NewPeerIds: []string{"peer2", "peer2"},
			})
			require.ErrorIs(t, err, types.ErrInvalidPostPayload)
		})

		t.Run("new_threshold of zero is rejected", func(t *testing.T) {
			_, err := k.UpdateRingPostByAcp(ctx, &types.MsgUpdateRingPostByAcp{
				Creator:       collab.Address,
				Namespace:     namespace,
				PostId:        postId,
				XNewThreshold: &types.MsgUpdateRingPostByAcp_NewThreshold{NewThreshold: 0},
			})
			require.ErrorIs(t, err, types.ErrInvalidPostPayload)
		})

		_, err = k.UpdateRingPostByAcp(ctx, &types.MsgUpdateRingPostByAcp{
			Creator:    collab.Address,
			Namespace:  namespace,
			PostId:     postId,
			NewPeerIds: []string{"peer2", "peer3"},
		})
		require.NoError(t, err)

		stored := k.getPost(ctx, namespaceId, postId)
		require.NotNil(t, stored)
		require.Equal(t, postId, stored.Id)

		parsed, err := parseRingPayloadJSON(stored.Payload)
		require.NoError(t, err)
		require.Equal(t, []string{"peer2", "peer3"}, *parsed.NewPeerIDs)
	})

	t.Run("new_threshold is updated when provided", func(t *testing.T) {
		threshold := uint32(3)
		_, err = k.UpdateRingPostByAcp(ctx, &types.MsgUpdateRingPostByAcp{
			Creator:       collab.Address,
			Namespace:     namespace,
			PostId:        postId,
			XNewThreshold: &types.MsgUpdateRingPostByAcp_NewThreshold{NewThreshold: threshold},
		})
		require.NoError(t, err)

		parsed, err := parseRingPayloadJSON(k.getPost(ctx, namespaceId, postId).Payload)
		require.NoError(t, err)
		require.Equal(t, threshold, *parsed.NewThreshold)
	})

	t.Run("pss_interval is updated when provided", func(t *testing.T) {
		interval := uint64(100)
		_, err = k.UpdateRingPostByAcp(ctx, &types.MsgUpdateRingPostByAcp{
			Creator:      collab.Address,
			Namespace:    namespace,
			PostId:       postId,
			XPssInterval: &types.MsgUpdateRingPostByAcp_PssInterval{PssInterval: interval},
		})
		require.NoError(t, err)

		parsed, err := parseRingPayloadJSON(k.getPost(ctx, namespaceId, postId).Payload)
		require.NoError(t, err)
		require.Equal(t, interval, *parsed.PSSInterval)
	})

	t.Run("unspecified fields are not changed", func(t *testing.T) {
		before, err := parseRingPayloadJSON(k.getPost(ctx, namespaceId, postId).Payload)
		require.NoError(t, err)
		originalPeerIDs := *before.PeerIDs

		_, err = k.UpdateRingPostByAcp(ctx, &types.MsgUpdateRingPostByAcp{
			Creator:       collab.Address,
			Namespace:     namespace,
			PostId:        postId,
			XNewThreshold: &types.MsgUpdateRingPostByAcp_NewThreshold{NewThreshold: 5},
		})
		require.NoError(t, err)

		after, err := parseRingPayloadJSON(k.getPost(ctx, namespaceId, postId).Payload)
		require.NoError(t, err)
		require.Equal(t, originalPeerIDs, *after.PeerIDs)
	})
}

func makeRingPayloadWithPolicy(t *testing.T, policyId string) []byte {
	t.Helper()
	payload := ringPayloadWithSeed(t, "sourcehub-bulletin-bls-test-seed-0001", []string{"peer1"}, 1, nil, nil)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(payload, &raw))
	raw["policy_id"] = policyId
	out, err := json.Marshal(raw)
	require.NoError(t, err)
	return out
}

func TestMsgCreatePost_DoesNotValidateRingPayload(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	ownerKey := secp256k1.GenPrivKey().PubKey()
	owner := authtypes.NewBaseAccount(sdk.AccAddress(ownerKey.Address()), ownerKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, owner)

	namespace := "orbis/rings/ring1"

	_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   owner.Address,
		Namespace: namespace,
	})
	require.NoError(t, err)

	_, err = k.CreatePost(ctx, &types.MsgCreatePost{
		Creator:   owner.Address,
		Namespace: namespace,
		Payload:   []byte(`{"ring_pk":"pk","peer_ids":["peer1"]}`),
	})
	require.NoError(t, err)
}


func TestMsgUpdateRingPostByThresholdSignature_ValidateBasic(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	ownerKey := secp256k1.GenPrivKey().PubKey()
	owner := authtypes.NewBaseAccount(sdk.AccAddress(ownerKey.Address()), ownerKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, owner)

	validMsg := &types.MsgUpdateRingPostByThresholdSignature{
		Creator:         owner.Address,
		Namespace:       "ns1",
		PostId:          "post1",
		SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL,
		Signature:       []byte("signature"),
	}
	require.NoError(t, validMsg.ValidateBasic())

	cases := []struct {
		msg     *types.MsgUpdateRingPostByThresholdSignature
		wantErr error
	}{
		{&types.MsgUpdateRingPostByThresholdSignature{}, sdkerrors.ErrInvalidAddress},
		{&types.MsgUpdateRingPostByThresholdSignature{Creator: owner.Address}, types.ErrInvalidNamespaceId},
		{&types.MsgUpdateRingPostByThresholdSignature{Creator: owner.Address, Namespace: "ns1"}, types.ErrInvalidPostId},
		{&types.MsgUpdateRingPostByThresholdSignature{Creator: owner.Address, Namespace: "ns1", PostId: "post1"}, types.ErrInvalidSignatureScheme},
		{&types.MsgUpdateRingPostByThresholdSignature{Creator: owner.Address, Namespace: "ns1", PostId: "post1", SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL}, types.ErrInvalidSignaturePayload},
	}
	for _, c := range cases {
		err := c.msg.ValidateBasic()
		require.ErrorIs(t, err, c.wantErr)
	}
}

func TestUpdateRingPostByThresholdSignature_ValidatesRingPayload(t *testing.T) {
	k, ctx := setupKeeper(t)
	ctx = ctx.WithChainID("sourcehub-test").WithBlockHeight(101)
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	ownerKey := secp256k1.GenPrivKey().PubKey()
	owner := authtypes.NewBaseAccount(sdk.AccAddress(ownerKey.Address()), ownerKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, owner)

	namespace := "ns1"
	namespaceId := getNamespaceId(namespace)
	originalPayload := ringPayloadWithSeed(
		t,
		"sourcehub-bulletin-bls-test-seed-0001",
		[]string{"peer1"},
		1,
		[]string{"peer2"},
		uint32Ptr(2),
	)

	_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   owner.Address,
		Namespace: namespace,
	})
	require.NoError(t, err)

	_, err = k.CreatePost(ctx, &types.MsgCreatePost{
		Creator:   owner.Address,
		Namespace: namespace,
		Payload:   originalPayload,
	})
	require.NoError(t, err)

	postId := types.GeneratePostId(namespaceId, originalPayload)

	invalidCurrentPayloads := [][]byte{
		[]byte(`not-json`),
		[]byte(`{"ring_pk":"pk","peer_ids":["peer1"]}`),
		[]byte(`{"ring_pk":"pk","peer_ids":["peer1"],"threshold":-1}`),
		[]byte(`{"ring_pk":"pk","peer_ids":["peer1"],"threshold":1}`),
	}

	for i, payload := range invalidCurrentPayloads {
		invalidPostID := fmt.Sprintf("invalid-post-%d", i)
		k.SetPost(ctx, types.Post{
			Id:         invalidPostID,
			Namespace:  namespaceId,
			CreatorDid: owner.Address,
			Payload:    payload,
		})

		_, err = k.UpdateRingPostByThresholdSignature(ctx, &types.MsgUpdateRingPostByThresholdSignature{
			Creator:         owner.Address,
			Namespace:       namespace,
			PostId:          invalidPostID,
			SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL,
		})
		require.ErrorIs(t, err, types.ErrInvalidPostPayload)

		stored := k.getPost(ctx, namespaceId, invalidPostID)
		require.NotNil(t, stored)
		require.Equal(t, payload, stored.Payload)
	}

	currentRingPayload, err := parseRingPayloadJSON(originalPayload)
	require.NoError(t, err)
	signDocFinalizedPayload, err := deriveFinalizedRingPayloadReshare(currentRingPayload)
	require.NoError(t, err)
	finalizedPayload, err := finalizeRingPayloadReshare(currentRingPayload, uint64(ctx.BlockHeight()))
	require.NoError(t, err)
	signature := signReshareFinalizePayload(
		t,
		ctx.ChainID(),
		namespace,
		postId,
		originalPayload,
		signDocFinalizedPayload,
		"sourcehub-bulletin-bls-test-seed-0001",
	)
	_, err = k.UpdateRingPostByThresholdSignature(ctx, &types.MsgUpdateRingPostByThresholdSignature{
		Creator:         owner.Address,
		Namespace:       namespace,
		PostId:          postId,
		SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL,
		Signature:       signature,
	})
	require.NoError(t, err)

	stored := k.getPost(ctx, namespaceId, postId)
	require.NotNil(t, stored)
	require.Equal(t, finalizedPayload, stored.Payload)
}

func TestUpdateRingPostByThresholdSignature_VerifiesThresholdSignature(t *testing.T) {
	k, ctx := setupKeeper(t)
	ctx = ctx.WithChainID("sourcehub-test").WithBlockHeight(202)
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	ownerKey := secp256k1.GenPrivKey().PubKey()
	owner := authtypes.NewBaseAccount(sdk.AccAddress(ownerKey.Address()), ownerKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, owner)

	updaterKey := secp256k1.GenPrivKey().PubKey()
	updater := authtypes.NewBaseAccount(sdk.AccAddress(updaterKey.Address()), updaterKey, 2, 1)
	k.accountKeeper.SetAccount(ctx, updater)

	namespace := "manual-ns"
	namespaceId := getNamespaceId(namespace)
	ownerDID, err := k.GetAcpKeeper().GetActorDID(ctx, owner.Address)
	require.NoError(t, err)
	k.SetNamespace(ctx, types.Namespace{
		Id:       namespaceId,
		OwnerDid: ownerDID,
		Creator:  owner.Address,
	})

	originalPayload := ringPayloadWithSeed(
		t,
		"sourcehub-bulletin-bls-test-seed-0001",
		[]string{"peer1"},
		1,
		[]string{"peer2"},
		uint32Ptr(2),
	)
	postId := "post1"
	k.SetPost(ctx, types.Post{
		Id:         postId,
		Namespace:  namespaceId,
		CreatorDid: ownerDID,
		Payload:    originalPayload,
	})
	require.Empty(t, k.GetPolicyId(ctx))

	currentRingPayload, err := parseRingPayloadJSON(originalPayload)
	require.NoError(t, err)
	signDocFinalizedPayload, err := deriveFinalizedRingPayloadReshare(currentRingPayload)
	require.NoError(t, err)
	finalizedPayload, err := finalizeRingPayloadReshare(currentRingPayload, uint64(ctx.BlockHeight()))
	require.NoError(t, err)
	signature := signReshareFinalizePayload(
		t,
		ctx.ChainID(),
		namespace,
		postId,
		originalPayload,
		signDocFinalizedPayload,
		"sourcehub-bulletin-bls-test-seed-0001",
	)
	tamperedSignature := append([]byte(nil), signature...)
	tamperedSignature[len(tamperedSignature)-1] ^= 0x01
	attackerSignature := signReshareFinalizePayload(
		t,
		ctx.ChainID(),
		namespace,
		postId,
		originalPayload,
		signDocFinalizedPayload,
		"sourcehub-bulletin-bls-test-seed-0002",
	)

	_, err = k.UpdateRingPostByThresholdSignature(ctx, &types.MsgUpdateRingPostByThresholdSignature{
		Creator:         updater.Address,
		Namespace:       namespace,
		PostId:          postId,
		SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL,
		Signature:       tamperedSignature,
	})
	require.ErrorIs(t, err, types.ErrInvalidThresholdSignature)

	stored := k.getPost(ctx, namespaceId, postId)
	require.NotNil(t, stored)
	require.Equal(t, originalPayload, stored.Payload)

	_, err = k.UpdateRingPostByThresholdSignature(ctx, &types.MsgUpdateRingPostByThresholdSignature{
		Creator:         updater.Address,
		Namespace:       namespace,
		PostId:          postId,
		SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL,
		Signature:       attackerSignature,
	})
	require.ErrorIs(t, err, types.ErrInvalidThresholdSignature)

	stored = k.getPost(ctx, namespaceId, postId)
	require.NotNil(t, stored)
	require.Equal(t, originalPayload, stored.Payload)

	_, err = k.UpdateRingPostByThresholdSignature(ctx, &types.MsgUpdateRingPostByThresholdSignature{
		Creator:         updater.Address,
		Namespace:       namespace,
		PostId:          postId,
		SignatureScheme: "bls12_381",
		Signature:       signature,
	})
	require.ErrorIs(t, err, types.ErrInvalidThresholdSignature)

	stored = k.getPost(ctx, namespaceId, postId)
	require.NotNil(t, stored)
	require.Equal(t, originalPayload, stored.Payload)

	post2ID := "post2"
	k.SetPost(ctx, types.Post{
		Id:         post2ID,
		Namespace:  namespaceId,
		CreatorDid: ownerDID,
		Payload:    originalPayload,
	})
	_, err = k.UpdateRingPostByThresholdSignature(ctx, &types.MsgUpdateRingPostByThresholdSignature{
		Creator:         updater.Address,
		Namespace:       namespace,
		PostId:          post2ID,
		SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL,
		Signature:       signature,
	})
	require.ErrorIs(t, err, types.ErrInvalidThresholdSignature)

	stored = k.getPost(ctx, namespaceId, post2ID)
	require.NotNil(t, stored)
	require.Equal(t, originalPayload, stored.Payload)

	staleCurrentPayload := ringPayloadWithSeed(
		t,
		"sourcehub-bulletin-bls-test-seed-0001",
		[]string{"peer1"},
		1,
		[]string{"peer2", "peer4"},
		uint32Ptr(2),
	)
	k.SetPost(ctx, types.Post{
		Id:         postId,
		Namespace:  namespaceId,
		CreatorDid: ownerDID,
		Payload:    staleCurrentPayload,
	})
	_, err = k.UpdateRingPostByThresholdSignature(ctx, &types.MsgUpdateRingPostByThresholdSignature{
		Creator:         updater.Address,
		Namespace:       namespace,
		PostId:          postId,
		SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL,
		Signature:       signature,
	})
	require.ErrorIs(t, err, types.ErrInvalidThresholdSignature)

	stored = k.getPost(ctx, namespaceId, postId)
	require.NotNil(t, stored)
	require.Equal(t, staleCurrentPayload, stored.Payload)

	staleNoncePayload := ringPayloadWithSeedAndNonce(
		t,
		"sourcehub-bulletin-bls-test-seed-0001",
		[]string{"peer1"},
		1,
		[]string{"peer2"},
		uint32Ptr(2),
		9,
	)
	k.SetPost(ctx, types.Post{
		Id:         postId,
		Namespace:  namespaceId,
		CreatorDid: ownerDID,
		Payload:    staleNoncePayload,
	})
	_, err = k.UpdateRingPostByThresholdSignature(ctx, &types.MsgUpdateRingPostByThresholdSignature{
		Creator:         updater.Address,
		Namespace:       namespace,
		PostId:          postId,
		SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL,
		Signature:       signature,
	})
	require.ErrorIs(t, err, types.ErrInvalidThresholdSignature)

	stored = k.getPost(ctx, namespaceId, postId)
	require.NotNil(t, stored)
	require.Equal(t, staleNoncePayload, stored.Payload)

	k.SetPost(ctx, types.Post{
		Id:         postId,
		Namespace:  namespaceId,
		CreatorDid: ownerDID,
		Payload:    originalPayload,
	})

	_, err = k.UpdateRingPostByThresholdSignature(ctx, &types.MsgUpdateRingPostByThresholdSignature{
		Creator:         updater.Address,
		Namespace:       namespace,
		PostId:          postId,
		SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL,
		Signature:       signature,
	})
	require.NoError(t, err)

	stored = k.getPost(ctx, namespaceId, postId)
	require.NotNil(t, stored)
	require.Equal(t, finalizedPayload, stored.Payload)

	_, err = k.UpdateRingPostByThresholdSignature(ctx, &types.MsgUpdateRingPostByThresholdSignature{
		Creator:         updater.Address,
		Namespace:       namespace,
		PostId:          postId,
		SignatureScheme: ThresholdSignatureSchemeBLS12381G1PKG2SigNUL,
		Signature:       signature,
	})
	require.ErrorIs(t, err, types.ErrInvalidPostPayload)
}

func signedRingPayload(t *testing.T, peerIDs []string, threshold uint32) ([]byte, []byte) {
	t.Helper()

	return signedRingPayloadWithSeed(t, "sourcehub-bulletin-bls-test-seed-0001", peerIDs, threshold)
}

func signedRingPayloadWithSeed(t *testing.T, seed string, peerIDs []string, threshold uint32) ([]byte, []byte) {
	t.Helper()

	payload := ringPayloadWithSeed(t, seed, peerIDs, threshold, nil, nil)
	signature := signPayloadWithSeed(t, seed, payload)

	return payload, signature
}

type testRingPayload struct {
	RingPK           string   `json:"ring_pk"`
	NewPeerIDs       []string `json:"new_peer_ids,omitempty"`
	NewThreshold     *uint32  `json:"new_threshold,omitempty"`
	PeerIDs          []string `json:"peer_ids"`
	Threshold        uint32   `json:"threshold"`
	PSSInterval      *uint64  `json:"pss_interval,omitempty"`
	BlockNumberNonce uint64   `json:"block_number_nonce"`
}

func ringPayloadWithSeed(
	t *testing.T,
	seed string,
	peerIDs []string,
	threshold uint32,
	newPeerIDs []string,
	newThreshold *uint32,
) []byte {
	t.Helper()

	return ringPayloadWithSeedAndNonce(t, seed, peerIDs, threshold, newPeerIDs, newThreshold, 0)
}

func ringPayloadWithSeedAndNonce(
	t *testing.T,
	seed string,
	peerIDs []string,
	threshold uint32,
	newPeerIDs []string,
	newThreshold *uint32,
	blockNumberNonce uint64,
) []byte {
	t.Helper()
	require.NotEmpty(t, peerIDs)

	secretKey := blst.KeyGen([]byte(seed))
	require.NotNil(t, secretKey)
	publicKey := new(blst.P1Affine).From(secretKey)

	payload, err := json.Marshal(testRingPayload{
		RingPK:           hex.EncodeToString(publicKey.Compress()),
		NewPeerIDs:       newPeerIDs,
		NewThreshold:     newThreshold,
		PeerIDs:          peerIDs,
		Threshold:        threshold,
		BlockNumberNonce: blockNumberNonce,
	})
	require.NoError(t, err)

	return payload
}

func signPayloadWithSeed(t *testing.T, seed string, payload []byte) []byte {
	t.Helper()

	secretKey := blst.KeyGen([]byte(seed))
	require.NotNil(t, secretKey)
	signature := new(blst.P2Affine).Sign(secretKey, payload, []byte(bls12381G2SignatureDST))

	return signature.Compress()
}

func signReshareFinalizePayload(
	t *testing.T,
	chainID string,
	namespace string,
	postID string,
	currentPayload []byte,
	finalizedPayload []byte,
	seed string,
) []byte {
	t.Helper()

	currentRingPayload, err := parseRingPayloadJSON(currentPayload)
	require.NoError(t, err)
	signBytes, err := ringReshareFinalizeSignBytes(chainID, namespace, postID, currentPayload, finalizedPayload, currentRingPayload)
	require.NoError(t, err)

	return signPayloadWithSeed(t, seed, signBytes)
}

func uint32Ptr(v uint32) *uint32 {
	return &v
}

// TestPolicy_CreatePostPermissionSurvives locks in that the `create_post`
// permission remains queryable via ACP with the `collaborator` relation.
// The keeper no longer gates CreatePost with this permission, but external
// consumers may still query it — this guards against accidental removal
// from the policy YAML.
func TestPolicy_CreatePostPermissionSurvives(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))

	ownerKey := secp256k1.GenPrivKey().PubKey()
	owner := authtypes.NewBaseAccount(sdk.AccAddress(ownerKey.Address()), ownerKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, owner)

	collabKey := secp256k1.GenPrivKey().PubKey()
	collab := authtypes.NewBaseAccount(sdk.AccAddress(collabKey.Address()), collabKey, 2, 1)
	k.accountKeeper.SetAccount(ctx, collab)

	setupTestPolicy(t, ctx, k)

	namespace := "ns-create-perm"
	namespaceId := getNamespaceId(namespace)

	_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   owner.Address,
		Namespace: namespace,
	})
	require.NoError(t, err)

	_, err = k.AddCollaborator(ctx, &types.MsgAddCollaborator{
		Creator:      owner.Address,
		Namespace:    namespace,
		Collaborator: collab.Address,
	})
	require.NoError(t, err)

	collabDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, collab.Address)
	require.NoError(t, err)

	allowed, err := hasPermission(ctx, &k, k.GetPolicyId(ctx), namespaceId, types.CreatePostPermission, collabDID, collab.Address)
	require.NoError(t, err)
	require.True(t, allowed, "create_post permission should resolve for a collaborator")

	outsiderKey := secp256k1.GenPrivKey().PubKey()
	outsider := authtypes.NewBaseAccount(sdk.AccAddress(outsiderKey.Address()), outsiderKey, 3, 1)
	k.accountKeeper.SetAccount(ctx, outsider)

	outsiderDID, err := k.GetAcpKeeper().IssueDIDFromAccountAddr(ctx, outsider.Address)
	require.NoError(t, err)

	allowed, err = hasPermission(ctx, &k, k.GetPolicyId(ctx), namespaceId, types.CreatePostPermission, outsiderDID, outsider.Address)
	require.NoError(t, err)
	require.False(t, allowed, "create_post permission should deny a non-collaborator")
}

func TestMsgServer(t *testing.T) {
	k, ctx := setupKeeper(t)
	require.NotNil(t, ctx)
	require.NotEmpty(t, k)
}

func TestMsgUpdateParams(t *testing.T) {
	k, ctx := setupKeeper(t)
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))
	wctx := sdk.UnwrapSDKContext(ctx)

	// default params
	testCases := []struct {
		name      string
		input     *types.MsgUpdateParams
		expErr    bool
		expErrMsg string
	}{
		{
			name: "invalid authority",
			input: &types.MsgUpdateParams{
				Authority: "invalid",
				Params:    params,
			},
			expErr:    true,
			expErrMsg: "invalid authority",
		},
		{
			name: "send enabled param",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    types.Params{},
			},
			expErr: false,
		},
		{
			name: "all good",
			input: &types.MsgUpdateParams{
				Authority: k.GetAuthority(),
				Params:    params,
			},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := k.UpdateParams(wctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgRegisterNamespace(t *testing.T) {
	k, ctx := setupKeeper(t)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)

	namespace := "ns1"

	testCases := []struct {
		name      string
		input     *types.MsgRegisterNamespace
		setup     func()
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "register namespace (error: invalid creator address)",
			input:     &types.MsgRegisterNamespace{},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid creator address",
		},
		{
			name: "register namespace (error: invalid namespace id)",
			input: &types.MsgRegisterNamespace{
				Creator: baseAcc.Address,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid namespace id",
		},
		{
			name: "register namespace (error: fetching capability for policy)",
			input: &types.MsgRegisterNamespace{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup: func() {
				k.SetPolicyId(ctx, "policy1")
			},
			expErr:    true,
			expErrMsg: "fetching capability for policy",
		},
		{
			name: "register namespace (error: fetching capability for invalid policy)",
			input: &types.MsgRegisterNamespace{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup: func() {
				k.SetPolicyId(ctx, "invalidPolicy1")
			},
			expErr:    true,
			expErrMsg: "fetching capability for policy invalidPolicy1",
		},
		{
			name: "register namespace (no error)",
			input: &types.MsgRegisterNamespace{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup: func() {
				setupTestPolicy(t, ctx, k)
			},
			expErr: false,
		},
		{
			name: "register namespace (error: namespace exists)",
			input: &types.MsgRegisterNamespace{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "namespace already exists",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.ValidateBasic()
			if err != nil {
				if tc.expErr {
					require.Contains(t, err.Error(), tc.expErrMsg)
					return
				}
				t.Fatalf("unexpected error in ValidateBasic: %v", err)
			}

			tc.setup()

			_, err = k.RegisterNamespace(ctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgCreatePost(t *testing.T) {
	k, ctx := setupKeeper(t)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)

	pubKey2 := secp256k1.GenPrivKey().PubKey()
	addr2 := sdk.AccAddress(pubKey2.Address())
	baseAcc2 := authtypes.NewBaseAccount(addr2, pubKey2, 2, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc2)

	namespace := "ns1"

	testCases := []struct {
		name      string
		input     *types.MsgCreatePost
		setup     func()
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "create post (error: invalid creator address)",
			input:     &types.MsgCreatePost{},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "nvalid creator address",
		},
		{
			name: "create post (error: invalid namespace id)",
			input: &types.MsgCreatePost{
				Creator: baseAcc.Address,
				Payload: []byte("post123"),
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid namespace id",
		},
		{
			name: "create post (error: no payload)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid post payload",
		},
		{
			name: "create post (error: namespace not found)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc.Address,
				Namespace: namespace,
				Payload:   []byte("post123"),
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "namespace not found",
		},
		{
			name: "create post (no error)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc.Address,
				Namespace: namespace,
				Payload:   []byte("post123"),
			},
			setup: func() {
				setupTestPolicy(t, ctx, k)

				_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
					Creator:   baseAcc.Address,
					Namespace: namespace,
				})
				require.NoError(t, err)
			},
			expErr: false,
		},
		{
			name: "create post (error: post already exists)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc.Address,
				Namespace: namespace,
				Payload:   []byte("post123"),
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "post already exists",
		},
		{
			name: "create post from unrelated signer (no error: posting is open)",
			input: &types.MsgCreatePost{
				Creator:   baseAcc2.Address,
				Namespace: namespace,
				Payload:   []byte("post-from-other"),
			},
			setup:  func() {},
			expErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.ValidateBasic()
			if err != nil {
				if tc.expErr {
					require.Contains(t, err.Error(), tc.expErrMsg)
					return
				}
				t.Fatalf("unexpected error in ValidateBasic: %v", err)
			}

			tc.setup()

			_, err = k.CreatePost(ctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgCreatePost_EmitsArtifactInEvent(t *testing.T) {
	k, ctx := setupKeeper(t)
	// given test policy and namespace
	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)
	setupTestPolicy(t, ctx, k)
	_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
		Creator:   baseAcc.Address,
		Namespace: "ns1",
	})
	require.NoError(t, err)

	// reset event manager
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	// when i create post
	post := types.MsgCreatePost{
		Creator:   baseAcc.Address,
		Namespace: "ns1",
		Payload:   []byte("some payload"),
		Artifact:  "session-id",
	}
	_, err = k.CreatePost(ctx, &post)

	// then post emit event with artifact
	require.NoError(t, err)

	evs := ctx.EventManager().Events()
	require.Len(t, evs, 1)

	creatorDid, err := k.acpKeeper.GetActorDID(ctx, baseAcc.Address)
	require.NoError(t, err)
	eventDid := "\"" + creatorDid + "\""
	ev := evs[0]
	require.Equal(t, `"session-id"`, ev.Attributes[0].Value)
	require.Equal(t, eventDid, ev.Attributes[1].Value)
	require.Equal(t, `"bulletin/ns1"`, ev.Attributes[2].Value)
}

func TestMsgAddCollaborator(t *testing.T) {
	k, ctx := setupKeeper(t)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)

	pubKey2 := secp256k1.GenPrivKey().PubKey()
	addr2 := sdk.AccAddress(pubKey2.Address())
	baseAcc2 := authtypes.NewBaseAccount(addr2, pubKey2, 2, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc2)

	pubKey3 := secp256k1.GenPrivKey().PubKey()
	addr3 := sdk.AccAddress(pubKey3.Address())
	baseAcc3 := authtypes.NewBaseAccount(addr3, pubKey3, 3, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc3)

	namespace := "ns1"

	testCases := []struct {
		name      string
		input     *types.MsgAddCollaborator
		setup     func()
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "add collaborator (error: invalid creator address)",
			input:     &types.MsgAddCollaborator{},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid creator address",
		},
		{
			name: "add collaborator (error: invalid namespace id)",
			input: &types.MsgAddCollaborator{
				Creator: baseAcc.Address,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid namespace id",
		},
		{
			name: "add collaborator (error: invalid collaborator address)",
			input: &types.MsgAddCollaborator{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid collaborator address",
		},
		{
			name: "add collaborator (error: invalid policy id)",
			input: &types.MsgAddCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid policy id",
		},
		{
			name: "add collaborator (error: namespace not found)",
			input: &types.MsgAddCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup: func() {
				k.SetPolicyId(ctx, "policy1")
			},
			expErr:    true,
			expErrMsg: "namespace not found",
		},
		{
			name: "add collaborator (no error)",
			input: &types.MsgAddCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup: func() {
				setupTestPolicy(t, ctx, k)

				_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
					Creator:   baseAcc.Address,
					Namespace: namespace,
				})
				require.NoError(t, err)
			},
			expErr: false,
		},
		{
			name: "add collaborator (error: collaborator already exists)",
			input: &types.MsgAddCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "collaborator already exists",
		},
		{
			name: "add collaborator (error: unauthorized)",
			input: &types.MsgAddCollaborator{
				Creator:      baseAcc2.Address,
				Collaborator: baseAcc3.Address,
				Namespace:    namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "actor is not a manager of relation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.ValidateBasic()
			if err != nil {
				if tc.expErr {
					require.Contains(t, err.Error(), tc.expErrMsg)
					return
				}
				t.Fatalf("unexpected error in ValidateBasic: %v", err)
			}

			tc.setup()

			_, err = k.AddCollaborator(ctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMsgRemoveCollaborator(t *testing.T) {
	k, ctx := setupKeeper(t)

	p := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, p))

	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())
	baseAcc := authtypes.NewBaseAccount(addr, pubKey, 1, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc)

	pubKey2 := secp256k1.GenPrivKey().PubKey()
	addr2 := sdk.AccAddress(pubKey2.Address())
	baseAcc2 := authtypes.NewBaseAccount(addr2, pubKey2, 2, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc2)

	pubKey3 := secp256k1.GenPrivKey().PubKey()
	addr3 := sdk.AccAddress(pubKey3.Address())
	baseAcc3 := authtypes.NewBaseAccount(addr3, pubKey3, 3, 1)
	k.accountKeeper.SetAccount(ctx, baseAcc3)

	namespace := "ns1"

	testCases := []struct {
		name      string
		input     *types.MsgRemoveCollaborator
		setup     func()
		expErr    bool
		expErrMsg string
	}{
		{
			name:      "remove collaborator (error: invalid creator address)",
			input:     &types.MsgRemoveCollaborator{},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid creator address",
		},
		{
			name: "remove collaborator (error: invalid namespace id)",
			input: &types.MsgRemoveCollaborator{
				Creator: baseAcc.Address,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid namespace id",
		},
		{
			name: "remove collaborator (error: invalid collaborator address)",
			input: &types.MsgRemoveCollaborator{
				Creator:   baseAcc.Address,
				Namespace: namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid collaborator address",
		},
		{
			name: "remove collaborator (error: invalid policy id)",
			input: &types.MsgRemoveCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "invalid policy id",
		},
		{
			name: "remove collaborator (error: namespace not found)",
			input: &types.MsgRemoveCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup: func() {
				k.SetPolicyId(ctx, "policy1")
			},
			expErr:    true,
			expErrMsg: "namespace not found",
		},
		{
			name: "remove collaborator (no error)",
			input: &types.MsgRemoveCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup: func() {
				setupTestPolicy(t, ctx, k)

				_, err := k.RegisterNamespace(ctx, &types.MsgRegisterNamespace{
					Creator:   baseAcc.Address,
					Namespace: namespace,
				})
				require.NoError(t, err)

				_, err = k.AddCollaborator(ctx, &types.MsgAddCollaborator{
					Creator:      baseAcc.Address,
					Collaborator: baseAcc2.Address,
					Namespace:    namespace,
				})
				require.NoError(t, err)
			},
			expErr: false,
		},
		{
			name: "remove collaborator (error: collaborator not found)",
			input: &types.MsgRemoveCollaborator{
				Creator:      baseAcc.Address,
				Collaborator: baseAcc2.Address,
				Namespace:    namespace,
			},
			setup:     func() {},
			expErr:    true,
			expErrMsg: "collaborator not found",
		},
		{
			name: "remove collaborator (error: unauthorized)",
			input: &types.MsgRemoveCollaborator{
				Creator:      baseAcc2.Address,
				Collaborator: baseAcc3.Address,
				Namespace:    namespace,
			},
			setup: func() {
				_, err := k.AddCollaborator(ctx, &types.MsgAddCollaborator{
					Creator:      baseAcc.Address,
					Collaborator: baseAcc2.Address,
					Namespace:    namespace,
				})
				require.NoError(t, err)

				_, err = k.AddCollaborator(ctx, &types.MsgAddCollaborator{
					Creator:      baseAcc.Address,
					Collaborator: baseAcc3.Address,
					Namespace:    namespace,
				})
				require.NoError(t, err)
			},
			expErr:    true,
			expErrMsg: "actor is not a manager of relation",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.ValidateBasic()
			if err != nil {
				if tc.expErr {
					require.Contains(t, err.Error(), tc.expErrMsg)
					return
				}
				t.Fatalf("unexpected error in ValidateBasic: %v", err)
			}

			tc.setup()

			_, err = k.RemoveCollaborator(ctx, tc.input)

			if tc.expErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
