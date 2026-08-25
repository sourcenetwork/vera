package access_ticket

import (
	"context"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/sourcenetwork/vera/x/acp/types"
)

// TicketIssuer is a responsible for generating Access Tickets
type TicketIssuer struct {
	abciService *abciService
	marshaler   Marshaler
	signer      signer
}

func NewTicketIssuer(abciService *abciService) TicketIssuer {
	return TicketIssuer{
		abciService: abciService,
		marshaler:   Marshaler{},
		signer:      signer{},
	}
}

// Issue fetches an Access Decision from its Id and generates an Access Ticket from it.
func (b *TicketIssuer) IssueRaw(ctx context.Context, decisionId string, privKey cryptotypes.PrivKey) (*types.AccessTicket, error) {
	decision, err := b.retrieveDecision(ctx, decisionId)
	if err != nil {
		return nil, err
	}

	proof, err := b.buildProof(ctx, decisionId)
	if err != nil {
		return nil, err
	}

	ticket := &types.AccessTicket{
		VersionDenominator: types.AccessTicketV1,
		DecisionId:         decisionId,
		Decision:           &decision,
		DecisionProof:      proof,
		Signature:          nil,
	}

	signature, err := b.signer.Sign(privKey, ticket)
	if err != nil {
		return nil, err
	}

	ticket.Signature = signature

	return ticket, nil
}

func (b *TicketIssuer) Issue(ctx context.Context, decisionId string, privKey cryptotypes.PrivKey) (string, error) {
	ticket, err := b.IssueRaw(ctx, decisionId, privKey)
	if err != nil {
		return "", err
	}

	return b.marshaler.Marshal(ticket)
}

func (b *TicketIssuer) buildProof(ctx context.Context, decisionId string) ([]byte, error) {
	decision, err := b.abciService.QueryDecision(ctx, decisionId, true, 0)
	if err != nil {
		return nil, err
	}

	bytes, err := decision.Marshal()
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// retrieveDecision uses an ABCI query to get the target decision object
func (b *TicketIssuer) retrieveDecision(ctx context.Context, decisionId string) (types.AccessDecision, error) {
	var decision types.AccessDecision

	// retrieve a decision without a proof at the latest height
	response, err := b.abciService.QueryDecision(ctx, decisionId, false, 0)
	if err != nil {
		return decision, err
	}

	err = decision.Unmarshal(response.Value)
	if err != nil {
		return decision, err
	}

	return decision, nil
}
