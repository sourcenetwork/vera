package sdk

import (
	"fmt"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	"github.com/sourcenetwork/vera/types"
)

var secp256k1MsgTypeUrl = cdctypes.MsgTypeURL(&secp256k1.PubKey{})

// TxSigner models an entity capable of providing signatures for a Tx.
//
// Effectively, it can be either a secp256k1 cosmos-sdk key or a pointer to a
// secp256k1 key in a cosmos-sdk like keyring.
type TxSigner interface {
	GetAccAddress() string
	GetPrivateKey() cryptotypes.PrivKey
}

// NewTxSignerFromKeyringKey receives a cosmos keyring and a named key in the keyring
// and returns a TxSigner capable of signing Txs.
// In order to sign Txs, the key must be of type secp256k1, as it's the only supported
// Tx signing key in CosmosSDK.
// See https://docs.cosmos.network/main/learn/beginner/accounts#keys-accounts-addresses-and-signatures
//
// Note: The adapter does not access the private key bytes directly, instead delegating
// the signing to the keyring itself. As such, any attempt to dump the bytes of the priv key
// will cause a panic
func NewTxSignerFromKeyringKey(keyring keyring.Keyring, name string) (TxSigner, error) {
	record, err := keyring.Key(name)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("key %v not found for keyring", name)
	}

	if record.PubKey.TypeUrl != secp256k1MsgTypeUrl {
		return nil, fmt.Errorf("cannot create signer from key %v: key must be of type secp256k1", name)
	}

	pubKey := &secp256k1.PubKey{}
	err = pubKey.Unmarshal(record.PubKey.Value)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal key %v: %v", name, err)
	}

	return &keyringTxSigner{
		keyring: keyring,
		keyname: name,
		pubkey:  pubKey,
	}, nil
}

// NewTxSignerFromAccountAddress takes a cosmos keyring and an account address
// and returns a TxSigner capable of signing Txs.
// If there are no keys matching the given address, returns an error.
//
// In order to sign Txs, the key must be of type secp256k1, as it's the only supported
// Tx signing key in CosmosSDK.
// See https://docs.cosmos.network/main/learn/beginner/accounts#keys-accounts-addresses-and-signatures
//
// Note: The adapter does not access the private key bytes directly, instead delegating
// the signing to the keyring itself. As such, any attempt to dump the bytes of the priv key
// will cause a panic
func NewTxSignerFromAccountAddress(keyring keyring.Keyring, address string) (TxSigner, error) {
	accAddr, err := types.AccAddressFromBech32(address)
	if err != nil {
		return nil, err
	}
	record, err := keyring.KeyByAddress(accAddr)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, fmt.Errorf("key %v not found for keyring", address)
	}

	if record.PubKey.TypeUrl != secp256k1MsgTypeUrl {
		return nil, fmt.Errorf("cannot create signer from key %v: key must be of type secp256k1", address)
	}

	pubKey := &secp256k1.PubKey{}
	err = pubKey.Unmarshal(record.PubKey.Value)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal key %v: %v", address, err)
	}

	return &keyringTxSigner{
		keyring: keyring,
		keyname: address,
		pubkey:  pubKey,
	}, nil
}

// keyringTxSigner wraps a keyring as a TxSigner
type keyringTxSigner struct {
	keyring keyring.Keyring
	keyname string
	pubkey  cryptotypes.PubKey
}

func (s *keyringTxSigner) GetAccAddress() string {
	addr := s.pubkey.Address().Bytes()
	return sdk.AccAddress(addr).String()
}

func (s *keyringTxSigner) GetPrivateKey() cryptotypes.PrivKey {
	return &keyringPKAdapter{
		keyring: s.keyring,
		keyname: s.keyname,
		pubkey:  s.pubkey,
	}
}

// keyringPKAdapter adapts a keyring + pubkey into a Cosmos PrivKey
// This type does not support the Bytes() method because it's not made to
// represent a handle to a Private Key.
// Calling Bytes() will cause a panic.
type keyringPKAdapter struct {
	keyring keyring.Keyring
	keyname string
	pubkey  cryptotypes.PubKey
}

func (a *keyringPKAdapter) Bytes() []byte {
	panic("dumping bytes from PrivKey in Keyring isn't supported")
}

func (a *keyringPKAdapter) Sign(msg []byte) ([]byte, error) {
	bytes, _, err := a.keyring.Sign(a.keyname, msg, signing.SignMode_SIGN_MODE_DIRECT)
	return bytes, err
}

func (a *keyringPKAdapter) PubKey() cryptotypes.PubKey {
	return a.pubkey
}

func (a *keyringPKAdapter) Equals(cryptotypes.LedgerPrivKey) bool {
	return false
}

func (a *keyringPKAdapter) Type() string {
	return secp256k1.PrivKeyName
}

func (a *keyringPKAdapter) Reset()         {}
func (a *keyringPKAdapter) ProtoMessage()  {}
func (a *keyringPKAdapter) String() string { return a.Type() }

// privKeySigner implements TxSigner for a Cosmos PrivKey
type privKeySigner struct {
	key cryptotypes.PrivKey
}

// TxSignerFromCosmosKey returns a TxSigner from a cosmos PrivKey
func TxSignerFromCosmosKey(priv cryptotypes.PrivKey) TxSigner {
	return &privKeySigner{
		key: priv,
	}
}

func (s *privKeySigner) GetPrivateKey() cryptotypes.PrivKey {
	return s.key
}

func (s *privKeySigner) GetAccAddress() string {
	addr := s.key.PubKey().Address().Bytes()
	return sdk.AccAddress(addr).String()
}
