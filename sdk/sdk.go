package sdk

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"cosmossdk.io/x/feegrant"
	cmtlog "github.com/cometbft/cometbft/libs/log"
	cometclient "github.com/cometbft/cometbft/rpc/client"
	"github.com/cometbft/cometbft/rpc/client/http"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	acptypes "github.com/sourcenetwork/sourcehub/x/acp/types"
	bulletintypes "github.com/sourcenetwork/sourcehub/x/bulletin/types"
)

// Opt configures the construction of a Client
type Opt func(*Client) error

// WithGRPCAddr sets the GRPC Address of a SourceHub node which the Client will connect to.
// If not specified defaults to DefaultGRPCAddr
// Note: Cosmos queries are not verifiable therefore only trusted RPC nodes should be used.
func WithGRPCAddr(addr string) Opt {
	return func(c *Client) error {
		c.grpcAddr = addr
		return nil
	}
}

// WithGRPCAddr sets the CometBFT RPC Address of a SourceHub node which the Client will connect to.
// If not set defaults to DefaultCometHTTPAddr
func WithCometRPCAddr(addr string) Opt {
	return func(c *Client) error {
		c.cometRPCAddr = addr
		return nil
	}
}

// WithGRPCOpts specifies the dial options which will be used to dial SourceHub's GRPC (queries) service
func WithGRPCOpts(opts ...grpc.DialOption) Opt {
	return func(c *Client) error {
		c.grpcOpts = opts
		return nil
	}
}

// NewClient returns a new SourceHub SDK client
func NewClient(opts ...Opt) (*Client, error) {
	client := &Client{
		grpcAddr:     DefaultGRPCAddr,
		cometRPCAddr: DefaultCometRPCAddr,
		logger:       cmtlog.NewTMLogger(os.Stderr),
	}

	for _, opt := range opts {
		opt(client)
	}

	//dialOpts := append()
	dialOpts := make([]grpc.DialOption, 0, len(client.grpcOpts)+1)
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	dialOpts = append(dialOpts, client.grpcOpts...)
	conn, err := grpc.Dial(client.grpcAddr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("sourcehub grpc dial: %w", err)
	}

	cometClient, err := http.New(client.cometRPCAddr, abciSocketPath)
	if err != nil {
		return nil, fmt.Errorf("comet rpc client: %w", err)
	}
	err = cometClient.Start()
	if err != nil {
		return nil, fmt.Errorf("starting comet rpc client: %w", err)
	}

	client.txClient = txtypes.NewServiceClient(conn)
	client.listener = NewTxListener(cometClient)
	client.conn = conn
	client.cometClient = cometClient

	return client, nil
}

// Client abstracts a set of connections to a SourceHub node.
// The Client type provides access to module specific clients and functionalities to
// interact with SourceHub such as performing module Queries (GRPC), Broadcast Txs
// and interact with CometBFT
type Client struct {
	grpcAddr     string
	cometRPCAddr string
	grpcOpts     []grpc.DialOption
	logger       cmtlog.Logger

	cometClient cometclient.Client
	conn        *grpc.ClientConn
	txClient    txtypes.ServiceClient
	listener    TxListener
}

// BroadcastTx broadcasts a signed Tx to a SourceHub node and returns the node's response.
// Callers can use TxResponse.TxHash to await or listen until the Tx is accepted and executed.
func (b *Client) BroadcastTx(ctx context.Context, tx xauthsigning.Tx) (*sdk.TxResponse, error) {
	encoder := authtx.DefaultTxEncoder()
	txBytes, err := encoder(tx)
	if err != nil {
		return nil, fmt.Errorf("marshaling tx: %w", err)
	}

	grpcRes, err := b.txClient.BroadcastTx(
		ctx,
		&txtypes.BroadcastTxRequest{
			Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
			TxBytes: txBytes,
		},
	)
	if err != nil {
		log.Printf("broadcasting err: %v", grpcRes)
		return nil, err
	}

	response := grpcRes.TxResponse

	//log.Printf("broadcast tx: %v", grpcRes)
	if response.Code != 0 {
		return response, fmt.Errorf("tx rejected: codespace %v: code %v: %v", response.Codespace, response.Code, response.RawLog)
	}

	return response, nil
}

// Close terminates the Client, freeing up resources and connections
func (b *Client) Close() {
	b.cometClient.Stop()
	b.conn.Close()
}

// ACPQueryClient returns a Query Client for the ACP module
func (c *Client) ACPQueryClient() acptypes.QueryClient {
	return acptypes.NewQueryClient(c.conn)
}

// BulletinQueryClient returns a Query Client for the Bulletin module
func (c *Client) BulletinQueryClient() bulletintypes.QueryClient {
	return bulletintypes.NewQueryClient(c.conn)
}

// BankQueryClient returns a Query Client for the Bank module
func (c *Client) BankQueryClient() banktypes.QueryClient {
	return banktypes.NewQueryClient(c.conn)
}

// FeegrantQueryClient returns a Query Client for the Feegrant module
func (c *Client) FeeGrantQueryClient() feegrant.QueryClient {
	return feegrant.NewQueryClient(c.conn)
}

// AuthQueryClient returns a Query Client for the Auth module
func (c *Client) AuthQueryClient() authtypes.QueryClient {
	return authtypes.NewQueryClient(c.conn)
}

// CometBFTRPCClient returns a Comet RPC Client
func (c *Client) CometBFTRPCClient() cometclient.Client {
	return c.cometClient
}

// TxListener returns a TxListener
func (c *Client) TxListener() TxListener {
	return c.listener
}

func (c *Client) ListenForTx(ctx context.Context, txHash string) <-chan *ListenResult {
	ch := make(chan *ListenResult)
	go func() {
		result, err := c.AwaitTx(ctx, txHash)
		ch <- newListenResult(result, err)
	}()

	return ch
}

func (c *Client) AwaitTx(ctx context.Context, txHash string) (*TxExecResult, error) {
	igniteClient, err := cosmosclient.New(ctx, cosmosclient.WithRPCClient(c.cometClient))
	if err != nil {
		return nil, err
	}
	result, err := igniteClient.WaitForTx(ctx, txHash)
	if err != nil {
		return nil, err
	}

	return newTxExecResult(result), nil
}

func (c *Client) GetTx(ctx context.Context, txHash string) (*TxExecResult, error) {
	bytes, err := hex.DecodeString(txHash)
	if err != nil {
		return nil, fmt.Errorf("invalid tx hash: %v", err)
	}

	txRPC, err := c.CometBFTRPCClient().Tx(ctx, bytes, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx: %v", err)
	}

	return newTxExecResult(txRPC), nil
}
