package access_ticket

import (
	"context"
	"fmt"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/rpc/client"
	"github.com/cometbft/cometbft/rpc/client/http"
	bfttypes "github.com/cometbft/cometbft/types"
)

const (
	abciSocketPath string = "/websocket"
)

func NewABCIService(addr string) (abciService, error) {
	client, err := http.New(addr, abciSocketPath)
	if err != nil {
		return abciService{}, fmt.Errorf("%w: %v", ErrExternal, err)
	}

	return abciService{
		addr:       addr,
		client:     client,
		keyBuilder: keyBuilder{},
	}, nil
}

// abciService performs an ABCI calls over a trusted node
type abciService struct {
	addr       string
	client     *http.HTTP
	keyBuilder keyBuilder
}

// Query a CometBFT node through the ABCI query method for an AccessDecision with decisionId.
// set prove true to return a query proof
// height corresponds to the height of the block at which the proof is required, set 0 to use the latest block
func (s *abciService) QueryDecision(ctx context.Context, decisionId string, prove bool, height int64) (*abcitypes.ResponseQuery, error) {
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
func (s *abciService) GetCurrentHeight(ctx context.Context) (int64, error) {
	resp, err := s.client.ABCIInfo(ctx)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrExternal, err)
	}

	return resp.Response.LastBlockHeight, nil
}

// GetBlockHeader returns the block header at the given height
func (s *abciService) GetBlockHeader(ctx context.Context, height int64) (bfttypes.Header, error) {
	resp, err := s.client.Block(ctx, &height)
	if err != nil {
		return bfttypes.Header{}, fmt.Errorf("%w: %v", ErrExternal, err)
	}

	return resp.Block.Header, nil
}
