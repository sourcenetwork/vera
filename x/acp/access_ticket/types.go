package access_ticket

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/sourcenetwork/sourcehub/x/acp/did"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

const separator = "."

// IAVL Store Querires expect the query to contain "key" as suffix to the Path
const iavlQuerySuffix = "key"

// Marshaler is responsible for marshaling and unamarshaling an AccessTicket to and from a string representation
type Marshaler struct{}

func (m *Marshaler) Marshal(ticket *types.AccessTicket) (string, error) {
	version := ticket.VersionDenominator
	ticketBytes, err := ticket.Marshal()
	if err != nil {
		return "", err
	}
	ticketEncoded := base64.URLEncoding.EncodeToString(ticketBytes)
	return version + separator + ticketEncoded, nil
}

func (m *Marshaler) Unmarshal(ticket string) (*types.AccessTicket, error) {

	version, encodedTicket, found := strings.Cut(ticket, separator)
	if !found {
		return nil, fmt.Errorf("invalid ticket: separator not found")
	}

	if version != types.AccessTicketV1 {
		return nil, fmt.Errorf("invalid ticket: invalid version")
	}

	ticketBytes, err := base64.URLEncoding.DecodeString(encodedTicket)
	if err != nil {
		return nil, fmt.Errorf("invalid ticket: encoded ticket: %v", err)

	}

	tkt := &types.AccessTicket{}
	err = tkt.Unmarshal(ticketBytes)
	if err != nil {
		return nil, fmt.Errorf("invalid ticket: encoded ticket: %v", err)
	}

	return tkt, nil
}

// keyBuilder builds keys to execute ABCI queries for AccessDecisions
type keyBuilder struct{}

// ABCIQueryKey returns the Key part to be used in an ABCIQuery
func (b *keyBuilder) ABCIQueryKey(decisionId string) []byte {
	return []byte(types.AccessDecisionRepositoryKeyPrefix + decisionId)
}

// KVKey returns the actual key used in the Cosmos app store.
// note this key contains the prefix from the root commit multistore and the IAVL store with prefixes
func (b *keyBuilder) KVKey(decisionId string) []byte {
	return []byte("/" + types.ModuleName + "/" + types.AccessDecisionRepositoryKeyPrefix + decisionId)
}

// ABCIQueryPath returns the Query Path for a query issued to the ACP module
func (b *keyBuilder) ABCIQueryPath() string {
	// Figuring out how to issue an ABCI query to a Cosmos app is a mess.
	// The request goes through to Tendermint and is sent straight to the application (ie cosmos base app),
	// it then goes through a multiple store layers, each with requirements for the key and none of which are documented.
	//
	// The entrpoint in baseapp itself.
	// The BaseApp can accept a set of prefixes and do different thigns with it,
	// for store state proofs it expected a "/store" prefix.
	// see cosmos/cosmos-sdk/baseapp/abci.go
	//
	// The request is then dispatched to the commit multi store.
	// The CMS dispatches the request to one of the substores using the substore name.
	// In our case the ACP module name.
	// see cosmos/cosmos-sdk/store/rootmulti/store.go
	//
	// It then goes to a IAVL store, the IAVL store expects keys to have a
	// "/key" suffix as part of the ABCI query path.
	// see cosmos/cosmos-sdk/store/iavl/store.go
	// IAVL is the last layer to process the Key field in the request. Now it's only the Data part.
	//
	// For the Data part it's necessary to figure out which prefix stores have been added to the mix but that's more straight forward.

	return "/" + baseapp.QueryPathStore + "/" + types.ModuleName + "/" + iavlQuerySuffix
}

func UnmarshalAndVerify(ctx context.Context, trustedNodeAddr string, ticketStr string) (*types.AccessTicket, error) {
	marshaler := Marshaler{}
	service, err := NewABCIService(trustedNodeAddr)
	if err != nil {
		return nil, err
	}
	ticketSpec := NewAccessTicketSpec(&service, &did.KeyResolver{})

	ticket, err := marshaler.Unmarshal(ticketStr)
	if err != nil {
		return nil, err
	}

	err = ticketSpec.SatisfiesRaw(ctx, ticket)
	if err != nil {
		return nil, err
	}

	return ticket, nil
}
