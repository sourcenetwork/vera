package access_ticket

import (
	"context"
	"fmt"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/rpc/client"
	"github.com/cometbft/cometbft/rpc/client/http"
	bfttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/sourcenetwork/sourcehub/x/acp/types"
)

const (
	abciSocketPath string = "/websocket"
)

// IAVL Store Querires expect the query to contain "key" as suffix to the Path
const iavlQuerySuffix = "key"

func NewABCIService(addr string) (ABCIService, error) {
	client, err := http.New(addr, abciSocketPath)
	if err != nil {
		return ABCIService{}, err
	}

	return ABCIService{
		addr:       addr,
		client:     client,
		keyBuilder: keyBuilder{},
	}, nil
}

// ABCIService performs an ABCI calls over a trusted node
type ABCIService struct {
	addr       string
	client     *http.HTTP
	keyBuilder keyBuilder
}

// Query a CometBFT node through the ABCI query method for an AccessDecision with decisionId.
// set prove true to return a query proof
// height corresponds to the height of the block at which the proof is required, set 0 to use the latest block
func (s *ABCIService) QueryDecision(ctx context.Context, decisionId string, prove bool, height int64) (*abcitypes.ResponseQuery, error) {
	opts := client.ABCIQueryOptions{
		Height: height,
		Prove:  prove,
	}
	path := s.keyBuilder.ABCIQueryPath()
	key := s.keyBuilder.ABCIQueryKey(decisionId)
	res, err := s.client.ABCIQueryWithOptions(ctx, path, key, opts)
	if err != nil {
		return nil, err
	}
	if res.Response.Value == nil {
		return nil, fmt.Errorf("decision %v: %w", decisionId, ErrDecisionNotFound)
	}

	return &res.Response, nil
}

// GetCurrentHeight returns the current height of a node
func (s *ABCIService) GetCurrentHeight(ctx context.Context) (uint64, error) {
	resp, err := s.client.ABCIInfo(ctx)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrExternal, err)
	}

	return uint64(resp.Response.LastBlockHeight), nil
}

// GetCurrentHeight returns the current height of a node
func (s *ABCIService) GetBlockHeader(ctx context.Context, height int64) (bfttypes.Header, error) {
	resp, err := s.client.Block(ctx, &height)
	if err != nil {
		return bfttypes.Header{}, fmt.Errorf("%w: %v", ErrExternal, err)
	}

	return resp.Block.Header, nil
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
