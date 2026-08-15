package sdk

const (
	Testnet1            = "vera-testnet-1"
	TestnetLatest       = Testnet1
	DefaultChainID      = TestnetLatest
	DefaultGRPCAddr     = "localhost:9090"
	DefaultCometRPCAddr = "tcp://localhost:26657"

	// abciSocketPath specifies the endpoint which the cometrpc http client connects to
	abciSocketPath string = "/websocket"
)
