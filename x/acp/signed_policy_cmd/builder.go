package signed_policy_cmd

import (
	"context"
	"crypto"
	"fmt"
	"strings"

	"github.com/cosmos/gogoproto/jsonpb"
	prototypes "github.com/cosmos/gogoproto/types"
	"github.com/go-jose/go-jose/v3"
	"github.com/go-jose/go-jose/v3/cryptosigner"

	"github.com/sourcenetwork/vera/x/acp/did"
	"github.com/sourcenetwork/vera/x/acp/types"
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
		return types.SignedPolicyCmdPayload{}, fmt.Errorf("cmdBuilder: Command invalid: %v", b.cmdErr)
	}

	if err := validatePolicyCmd(b.cmd.Cmd); err != nil {
		return types.SignedPolicyCmdPayload{}, fmt.Errorf("cmdBuilder: Command invalid: %v", err)
	}

	return b.cmd, nil
}

// validatePolicyCmd performs basic structural validation on the embedded PolicyCmd.
// It ensures required fields are present and non-empty for each command variant.
func validatePolicyCmd(cmd *types.PolicyCmd) error {
	switch c := cmd.GetCmd().(type) {
	case *types.PolicyCmd_SetRelationshipCmd:
		rel := c.SetRelationshipCmd.GetRelationship()
		if rel == nil {
			return fmt.Errorf("set_relationship: relationship is required")
		}
		obj := rel.GetObject()
		if obj == nil || strings.TrimSpace(obj.GetResource()) == "" || strings.TrimSpace(obj.GetId()) == "" {
			return fmt.Errorf("set_relationship: object resource and id are required")
		}
		if strings.TrimSpace(rel.GetRelation()) == "" {
			return fmt.Errorf("set_relationship: relation is required")
		}
		subj := rel.GetSubject()
		if subj == nil {
			return fmt.Errorf("set_relationship: subject is required")
		}
		validSubject := false
		if a := subj.GetActor(); a != nil && strings.TrimSpace(a.GetId()) != "" {
			validSubject = true
		}
		if o := subj.GetObject(); o != nil && strings.TrimSpace(o.GetResource()) != "" && strings.TrimSpace(o.GetId()) != "" {
			validSubject = true
		}
		if as := subj.GetActorSet(); as != nil && strings.TrimSpace(as.GetRelation()) != "" {
			validSubject = true
		}
		if subj.GetAllActors() != nil {
			validSubject = true
		}
		if !validSubject {
			return fmt.Errorf("set_relationship: invalid subject")
		}
		return nil

	case *types.PolicyCmd_DeleteRelationshipCmd:
		rel := c.DeleteRelationshipCmd.GetRelationship()
		if rel == nil {
			return fmt.Errorf("delete_relationship: relationship is required")
		}
		obj := rel.GetObject()
		if obj == nil || strings.TrimSpace(obj.GetResource()) == "" || strings.TrimSpace(obj.GetId()) == "" {
			return fmt.Errorf("delete_relationship: object resource and id are required")
		}
		if strings.TrimSpace(rel.GetRelation()) == "" {
			return fmt.Errorf("delete_relationship: relation is required")
		}
		if rel.GetSubject() == nil {
			return fmt.Errorf("delete_relationship: subject is required")
		}
		return nil

	case *types.PolicyCmd_RegisterObjectCmd:
		obj := c.RegisterObjectCmd.GetObject()
		if obj == nil || strings.TrimSpace(obj.GetResource()) == "" || strings.TrimSpace(obj.GetId()) == "" {
			return fmt.Errorf("register_object: object resource and id are required")
		}
		return nil

	case *types.PolicyCmd_ArchiveObjectCmd:
		obj := c.ArchiveObjectCmd.GetObject()
		if obj == nil || strings.TrimSpace(obj.GetResource()) == "" || strings.TrimSpace(obj.GetId()) == "" {
			return fmt.Errorf("archive_object: object resource and id are required")
		}
		return nil

	case *types.PolicyCmd_UnarchiveObjectCmd:
		obj := c.UnarchiveObjectCmd.GetObject()
		if obj == nil || strings.TrimSpace(obj.GetResource()) == "" || strings.TrimSpace(obj.GetId()) == "" {
			return fmt.Errorf("unarchive_object: object resource and id are required")
		}
		return nil

	case *types.PolicyCmd_CommitRegistrationsCmd:
		if len(c.CommitRegistrationsCmd.GetCommitment()) == 0 {
			return fmt.Errorf("commit_registrations: commitment is required")
		}
		return nil

	case *types.PolicyCmd_RevealRegistrationCmd:
		if c.RevealRegistrationCmd.GetProof() == nil {
			return fmt.Errorf("reveal_registration: proof is required")
		}
		if c.RevealRegistrationCmd.GetRegistrationsCommitmentId() == 0 {
			return fmt.Errorf("reveal_registration: registrations_commitment_id must be > 0")
		}
		return nil

	case *types.PolicyCmd_FlagHijackAttemptCmd:
		if c.FlagHijackAttemptCmd.GetEventId() == 0 {
			return fmt.Errorf("flag_hijack_attempt: event_id must be > 0")
		}
		return nil

	default:
		return fmt.Errorf("unknown command variant")
	}
}

// IssuedAt sets the creation timestamp
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
