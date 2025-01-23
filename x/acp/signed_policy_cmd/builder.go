package signed_policy_cmd

import (
	"context"
	"crypto"
	"fmt"

	"github.com/cosmos/gogoproto/jsonpb"
	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/cryptosigner"

	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func NewCmdBuilder(clock LogicalClock, params types.Params) *CmdBuilder {
	return &CmdBuilder{
		clock:  clock,
		params: params,
	}
}

// CmdBuilder builds PolicyCmdPayloads
type CmdBuilder struct {
	clock  LogicalClock
	cmd    types.SignedPolicyCmdPayload
	params types.Params
	cmdErr error
	signer crypto.Signer
}

// BuildJWS produces a signed JWS for the specified Cmd
func (b *CmdBuilder) BuildJWS(ctx context.Context) (string, error) {
	if b.signer == nil {
		return "", fmt.Errorf("cmdBuilder failed: %w", ErrSignerRequired)
	}

	payload, err := b.Build(ctx)
	if err != nil {
		return "", err
	}

	return SignPayload(payload, b.signer)
}

// SetSigner sets the Signer for the Builder, which will be used to produce a JWS
func (b *CmdBuilder) SetSigner(signer crypto.Signer) {
	b.signer = signer
}

// GetSigner returns the currently set Signer
func (b *CmdBuilder) GetSigner() crypto.Signer {
	return b.signer
}

// Build validates the data provided to the Builder, validates it and returns a SignedPolicyCmdPayload or an error.
func (b *CmdBuilder) Build(ctx context.Context) (types.SignedPolicyCmdPayload, error) {
	height, err := b.clock.GetTimestampNow(ctx)
	if err != nil {
		return types.SignedPolicyCmdPayload{}, fmt.Errorf("cmdBuilder: timestamp failed: %v", err)
	}

	b.cmd.IssuedHeight = height

	if b.cmd.IssuedAt == nil {
		b.cmd.IssuedAt = prototypes.TimestampNow()
	}

	if b.cmd.ExpirationDelta == 0 {
		b.cmd.ExpirationDelta = b.params.PolicyCommandMaxExpirationDelta
	}

	if b.cmd.PolicyId == "" {
		return types.SignedPolicyCmdPayload{}, fmt.Errorf("cmdBuilder: policy id: %w", ErrBuilderMissingArgument)
	}

	if b.cmd.ExpirationDelta > b.params.PolicyCommandMaxExpirationDelta {
		return types.SignedPolicyCmdPayload{}, fmt.Errorf("cmdBuilder: %v", ErrExpirationDeltaTooLarge)
	}

	if err := did.IsValidDID(b.cmd.Actor); err != nil {
		return types.SignedPolicyCmdPayload{}, fmt.Errorf("cmdBuilder: invalid actor: %v", err)
	}

	if b.cmd.Cmd == nil {
		return types.SignedPolicyCmdPayload{}, fmt.Errorf("cmdBuilder: Command not specified: %v", ErrBuilderMissingArgument)
	}

	if b.cmdErr != nil {
		// TODO validate commands
		return types.SignedPolicyCmdPayload{}, fmt.Errorf("cmdBuilder: Command invalid: %v", b.cmdErr)
	}

	return b.cmd, nil
}

// CreationTimestamp sets the creation timestamp
func (b *CmdBuilder) IssuedAt(ts *prototypes.Timestamp) {
	b.cmd.IssuedAt = ts
}

// Actor sets the Actor for the Command
func (b *CmdBuilder) Actor(did string) {
	b.cmd.Actor = did
}

// ExpirationDelta specifies the number of blocks after the issue height for which the Command will be valid.
func (b *CmdBuilder) ExpirationDelta(delta uint64) {
	b.cmd.ExpirationDelta = delta
}

// PolicyID sets the Policy ID for the payload
func (b *CmdBuilder) PolicyID(id string) {
	b.cmd.PolicyId = id
}

// PolicyCmd sets the command to be issued with the Signed token
func (b *CmdBuilder) PolicyCmd(cmd *types.PolicyCmd) {
	b.cmd.Cmd = cmd
}

// SignPayload produces a JWS serialized version of a Payload from a signing key
func SignPayload(cmd types.SignedPolicyCmdPayload, skey crypto.Signer) (string, error) {
	marshaler := jsonpb.Marshaler{}
	payload, err := marshaler.MarshalToString(&cmd)
	if err != nil {
		return "", err
	}

	opaque := cryptosigner.Opaque(skey)
	key := jose.SigningKey{
		Algorithm: opaque.Algs()[0],
		Key:       opaque,
	}
	var opts *jose.SignerOptions
	signer, err := jose.NewSigner(key, opts)
	if err != nil {
		return "", err
	}

	obj, err := signer.Sign([]byte(payload))
	if err != nil {
		return "", err
	}

	return obj.FullSerialize(), nil
}
