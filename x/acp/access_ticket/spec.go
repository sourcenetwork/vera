package access_ticket

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	"github.com/TBD54566975/ssi-sdk/crypto"
	"github.com/TBD54566975/ssi-sdk/did/key"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/merkle"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"

	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

func NewAccessTicketSpec(serv *abciService, resolver did.Resolver) AccessTicketSpec {
	return AccessTicketSpec{
		abciService: serv,
		resolver:    resolver,
		marshaler:   Marshaler{},
	}
}

type AccessTicketSpec struct {
	abciService *abciService
	resolver    did.Resolver
	keyBuilder  keyBuilder
	signer      signer
	marshaler   Marshaler
}

func (s *AccessTicketSpec) Satisfies(ctx context.Context, ticket string) error {
	tkt, err := s.marshaler.Unmarshal(ticket)
	if err != nil {
		return err
	}
	return s.SatisfiesRaw(ctx, tkt)
}

// Satisfies inspects an AccessTicket and verifies whether it meets specification and is valid.
// Returns a non-nil error if the Ticket is invalid
func (s *AccessTicketSpec) SatisfiesRaw(ctx context.Context, ticket *types.AccessTicket) error {
	err := s.verifyProof(ctx, ticket.DecisionId, ticket.DecisionProof)
	if err != nil {
		return err
	}

	didStr := ticket.Decision.Actor

	did := key.DIDKey(didStr)
	pubBytes, _, keytype, err := did.Decode()
	if err != nil {
		return err
	}
	if keytype != crypto.SECP256k1 {
		return fmt.Errorf("unsupported key type: expected SECP256k1: got %v", keytype)
	}
	pkey := &secp256k1.PubKey{
		Key: pubBytes,
	}

	err = s.signer.Verify(pkey, ticket)
	if err != nil {
		return err
	}

	err = s.validateDecision(ticket)
	if err != nil {
		return err
	}

	heightInt, err := s.abciService.GetCurrentHeight(ctx)
	height := uint64(heightInt)
	if err != nil {
		return err
	}

	decisionExpiration := s.computeDecisionExpiration(ticket.Decision)
	proofExpiration := s.computeProofExpiration(ticket.Decision)
	ticketExpiration := s.computeTicketExpiration(ticket.Decision)

	if height > decisionExpiration {
		return ErrExpiredDecision
	}
	if height > proofExpiration {
		return ErrExpiredDecision
	}
	if height > ticketExpiration {
		return ErrExpiredTicket
	}

	return nil
}

func (s *AccessTicketSpec) computeDecisionExpiration(decision *types.AccessDecision) uint64 {
	return decision.IssuedHeight + decision.Params.DecisionExpirationDelta
}

func (s *AccessTicketSpec) computeTicketExpiration(decision *types.AccessDecision) uint64 {
	return decision.IssuedHeight + decision.Params.TicketExpirationDelta
}

func (s *AccessTicketSpec) computeProofExpiration(decision *types.AccessDecision) uint64 {
	return decision.IssuedHeight + decision.Params.ProofExpirationDelta
}

func (s *AccessTicketSpec) verifyProof(ctx context.Context, decisionId string, decisionProof []byte) error {
	abciQuery := &abcitypes.ResponseQuery{}
	err := abciQuery.Unmarshal(decisionProof)
	if err != nil {
		return ErrInvalidDecisionProof
	}

	height := abciQuery.Height + 1
	header, err := s.abciService.GetBlockHeader(ctx, height)
	if err != nil {
		return err
	}

	root := header.AppHash.Bytes()
	key := string(s.keyBuilder.KVKey(decisionId))

	runtime := merkle.NewProofRuntime()
	runtime.RegisterOpDecoder(storetypes.ProofOpIAVLCommitment, storetypes.CommitmentOpDecoder)
	runtime.RegisterOpDecoder(storetypes.ProofOpSimpleMerkleCommitment, storetypes.CommitmentOpDecoder)
	err = runtime.VerifyValue(abciQuery.ProofOps, root, key, abciQuery.Value)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidDecisionProof, err)
	}

	return nil
}

func (s *AccessTicketSpec) validateDecision(ticket *types.AccessTicket) error {
	decisionHash := ticket.Decision.ProduceId()

	if decisionHash != ticket.DecisionId {
		return fmt.Errorf("expected %v got %v: %w", ticket.DecisionId, decisionHash, ErrDecisionTampered)
	}

	if decisionHash != ticket.Decision.Id {
		return fmt.Errorf("expected %v got %v: %w", ticket.DecisionId, decisionHash, ErrDecisionTampered)
	}

	return nil
}
