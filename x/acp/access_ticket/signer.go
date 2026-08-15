package access_ticket

import (
	"crypto/sha256"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/sourcenetwork/vera/x/acp/types"
)

// signer is responsible for generating a Ticket signature and verifying it
type signer struct{}

func (s *signer) Sign(key cryptotypes.PrivKey, ticket *types.AccessTicket) ([]byte, error) {
	digest := s.hashTicket(ticket)
	return key.Sign(digest)
}

func (s *signer) Verify(key cryptotypes.PubKey, ticket *types.AccessTicket) error {
	digest := s.hashTicket(ticket)
	ok := key.VerifySignature(digest, ticket.Signature)
	if !ok {
		return ErrInvalidSignature
	}
	return nil
}

// hashTicket produces a sha256 which uniquely identifies a Ticket.
func (b *signer) hashTicket(ticket *types.AccessTicket) []byte {
	hasher := sha256.New()

	hasher.Write([]byte(ticket.VersionDenominator))
	// NOTE the Decision ID is produced as a Hash of all fields in it.
	// It's paramount that this invariant is not violated to ensure the
	// safety of the protocol
	hasher.Write([]byte(ticket.DecisionId))
	hasher.Write(ticket.DecisionProof)

	return hasher.Sum(nil)
}
