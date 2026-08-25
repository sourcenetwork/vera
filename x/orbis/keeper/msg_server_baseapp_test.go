package keeper

import (
	"encoding/hex"
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	txsigning "cosmossdk.io/x/tx/signing"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	orbistypes "github.com/sourcenetwork/vera/x/orbis/types"
)

func TestMsgServer_FinalizeRing_PkConflictDeletesRingThroughBaseApp(t *testing.T) {
	registry, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: txsigning.Options{
			AddressCodec:          addresscodec.NewBech32Codec("cosmos"),
			ValidatorAddressCodec: addresscodec.NewBech32Codec("cosmosvaloper"),
		},
	})
	require.NoError(t, err)
	authtypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	orbistypes.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	orbisStoreKey := storetypes.NewKVStoreKey(orbistypes.StoreKey)
	authStoreKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	bApp := baseapp.NewBaseApp(
		t.Name(),
		log.NewNopLogger(),
		dbm.NewMemDB(),
		txConfig.TxDecoder(),
	)
	bApp.SetInterfaceRegistry(registry)
	bApp.MsgServiceRouter().SetInterfaceRegistry(registry)
	bApp.MountStores(orbisStoreKey, authStoreKey)
	bApp.SetAnteHandler(func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	})
	require.NoError(t, bApp.LoadLatestVersion())

	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(authStoreKey),
		authtypes.ProtoBaseAccount,
		map[string][]string{authtypes.FeeCollectorName: nil},
		authcodec.NewBech32Codec("cosmos"),
		"cosmos",
		authority.String(),
	)
	orbisKeeper := NewKeeper(
		cdc,
		runtime.NewKVStoreService(orbisStoreKey),
		log.NewNopLogger(),
		authority.String(),
		accountKeeper,
		nil,
		nil,
	)
	orbistypes.RegisterMsgServer(bApp.MsgServiceRouter(), &orbisKeeper)

	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1})
	require.NoError(t, err)

	ctx := bApp.NewContextLegacy(false, cmtproto.Header{Height: 1})
	peer1PrivKey := secp256k1.GenPrivKeyFromSecret([]byte("peer-1"))
	peer2PrivKey := secp256k1.GenPrivKeyFromSecret([]byte("peer-2"))
	peer1NodeKey := hex.EncodeToString(peer1PrivKey.PubKey().Bytes())
	peer2NodeKey := hex.EncodeToString(peer2PrivKey.PubKey().Bytes())
	peer2Address := sdk.AccAddress(peer2PrivKey.PubKey().Address())

	peer2Account := accountKeeper.NewAccountWithAddress(ctx, peer2Address)
	require.NoError(t, peer2Account.SetPubKey(peer2PrivKey.PubKey()))
	accountKeeper.SetAccount(ctx, peer2Account)

	const ringID = "ring-with-conflicting-finalizations"
	orbisKeeper.SetRing(ctx, orbistypes.Ring{
		Id:           ringID,
		PeerNodeKeys: []string{peer1NodeKey, peer2NodeKey},
		Threshold:    2,
		PolicyId:     "policy",
		Reporting:    orbistypes.DefaultReportingConfig(),
		Confirmations: []*orbistypes.RingConfirmation{
			{
				NodeKey: peer1NodeKey,
				RingPk:  "pk-version-A",
			},
		},
	})
	require.NotNil(t, orbisKeeper.GetRing(ctx, ringID))

	txBuilder := txConfig.NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(&orbistypes.MsgFinalizeRing{
		Creator: peer2Address.String(),
		RingId:  ringID,
		RingPk:  "pk-version-B",
	}))

	_, result, err := bApp.SimDeliver(txConfig.TxEncoder(), txBuilder.GetTx())
	require.NoError(t, err)
	require.Len(t, result.MsgResponses, 1)
	var finalizeResp orbistypes.MsgFinalizeRingResponse
	require.NoError(t, cdc.Unmarshal(result.MsgResponses[0].Value, &finalizeResp))
	require.Equal(t, orbistypes.FinalizeRingOutcome_CONFLICT_DELETED, finalizeResp.Outcome)
	require.Nil(t, orbisKeeper.GetRing(ctx, ringID))

	var ringDeletedEvent *orbistypes.EventRingDeleted
	for _, event := range result.Events {
		if event.Type != "vera.orbis.EventRingDeleted" {
			continue
		}
		typedEvent, err := sdk.ParseTypedEvent(event)
		require.NoError(t, err)
		ringDeletedEvent = typedEvent.(*orbistypes.EventRingDeleted)
		break
	}
	require.Equal(t, &orbistypes.EventRingDeleted{
		RingId: ringID,
		Reason: "ring_pk_conflict",
	}, ringDeletedEvent)
}

func TestMsgServer_CancelPendingRing_DeletesRingThroughBaseApp(t *testing.T) {
	registry, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: txsigning.Options{
			AddressCodec:          addresscodec.NewBech32Codec("cosmos"),
			ValidatorAddressCodec: addresscodec.NewBech32Codec("cosmosvaloper"),
		},
	})
	require.NoError(t, err)
	authtypes.RegisterInterfaces(registry)
	cryptocodec.RegisterInterfaces(registry)
	orbistypes.RegisterInterfaces(registry)

	cdc := codec.NewProtoCodec(registry)
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)

	orbisStoreKey := storetypes.NewKVStoreKey(orbistypes.StoreKey)
	authStoreKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	bApp := baseapp.NewBaseApp(
		t.Name(),
		log.NewNopLogger(),
		dbm.NewMemDB(),
		txConfig.TxDecoder(),
	)
	bApp.SetInterfaceRegistry(registry)
	bApp.MsgServiceRouter().SetInterfaceRegistry(registry)
	bApp.MountStores(orbisStoreKey, authStoreKey)
	bApp.SetAnteHandler(func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	})
	require.NoError(t, bApp.LoadLatestVersion())

	authority := authtypes.NewModuleAddress(govtypes.ModuleName)
	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(authStoreKey),
		authtypes.ProtoBaseAccount,
		map[string][]string{authtypes.FeeCollectorName: nil},
		authcodec.NewBech32Codec("cosmos"),
		"cosmos",
		authority.String(),
	)
	orbisKeeper := NewKeeper(
		cdc,
		runtime.NewKVStoreService(orbisStoreKey),
		log.NewNopLogger(),
		authority.String(),
		accountKeeper,
		nil,
		nil,
	)
	orbistypes.RegisterMsgServer(bApp.MsgServiceRouter(), &orbisKeeper)

	_, err = bApp.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1})
	require.NoError(t, err)

	ctx := bApp.NewContextLegacy(false, cmtproto.Header{Height: 1})
	peerPrivKey := secp256k1.GenPrivKeyFromSecret([]byte("cancelling-peer"))
	peerNodeKey := hex.EncodeToString(peerPrivKey.PubKey().Bytes())
	peerAddress := sdk.AccAddress(peerPrivKey.PubKey().Address())

	peerAccount := accountKeeper.NewAccountWithAddress(ctx, peerAddress)
	require.NoError(t, peerAccount.SetPubKey(peerPrivKey.PubKey()))
	accountKeeper.SetAccount(ctx, peerAccount)

	const ringID = "unfinished-ring-to-cancel"
	orbisKeeper.SetRing(ctx, orbistypes.Ring{
		Id:           ringID,
		CreatorDid:   "did:example:ring-creator",
		PeerNodeKeys: []string{peerNodeKey},
		Threshold:    1,
		PolicyId:     "policy",
		Reporting:    orbistypes.DefaultReportingConfig(),
	})
	require.NotNil(t, orbisKeeper.GetRing(ctx, ringID))

	txBuilder := txConfig.NewTxBuilder()
	require.NoError(t, txBuilder.SetMsgs(&orbistypes.MsgCancelPendingRing{
		Creator: peerAddress.String(),
		RingId:  ringID,
	}))

	_, result, err := bApp.SimDeliver(txConfig.TxEncoder(), txBuilder.GetTx())
	require.NoError(t, err)
	require.Len(t, result.MsgResponses, 1)
	var cancelResp orbistypes.MsgCancelPendingRingResponse
	require.NoError(t, cdc.Unmarshal(result.MsgResponses[0].Value, &cancelResp))
	require.Nil(t, orbisKeeper.GetRing(ctx, ringID))

	var ringDeletedEvent *orbistypes.EventRingDeleted
	for _, event := range result.Events {
		if event.Type != "vera.orbis.EventRingDeleted" {
			continue
		}
		typedEvent, err := sdk.ParseTypedEvent(event)
		require.NoError(t, err)
		ringDeletedEvent = typedEvent.(*orbistypes.EventRingDeleted)
		break
	}
	require.Equal(t, &orbistypes.EventRingDeleted{
		RingId: ringID,
		Reason: "dkg_cancelled",
	}, ringDeletedEvent)
}
