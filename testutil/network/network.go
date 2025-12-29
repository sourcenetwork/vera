package network

import (
	"fmt"
	"os"
	"strings"
	"testing"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"

	"github.com/sourcenetwork/sourcehub/app"
	"github.com/sourcenetwork/sourcehub/app/params"
)

type (
	Network = network.Network
	Config  = network.Config
)

// NetworkOptions contains options for configuring the network.
type NetworkOptions struct {
	EnableFaucet bool
}

// testAppOptions creates app options for testing with optional faucet configuration.
type testAppOptions struct {
	enableFaucet bool
	v            *viper.Viper
}

// NewTestAppOptions creates test app options with faucet configuration.
func NewTestAppOptions(enableFaucet bool) *testAppOptions {
	v := viper.New()
	v.Set("faucet.enable_faucet", enableFaucet)
	return &testAppOptions{
		enableFaucet: enableFaucet,
		v:            v,
	}
}

// Get implements servertypes.AppOptions interface.
func (opts *testAppOptions) Get(key string) interface{} {
	return opts.v.Get(key)
}

// New creates instance with fully configured cosmos network.
// Accepts optional config, that will be used in place of the DefaultConfig() if provided.
func New(t *testing.T, configs ...Config) *Network {
	return NewWithOptions(t, NetworkOptions{EnableFaucet: false}, configs...)
}

// NewWithOptions creates instance with fully configured cosmos network and custom options.
// Accepts optional config and options, that will be used in place of the DefaultConfig() if provided.
func NewWithOptions(t *testing.T, options NetworkOptions, configs ...Config) *Network {
	t.Helper()
	if len(configs) > 1 {
		panic("at most one config should be provided")
	}
	var cfg network.Config
	if len(configs) == 0 {
		cfg = DefaultConfigWithOptions(options)
	} else {
		cfg = configs[0]
	}
	baseDir, err := os.MkdirTemp("", t.Name())
	require.NoError(t, err)
	net, err := network.New(t, baseDir, cfg)
	require.NoError(t, err)
	val := net.Validators[0]
	if options.EnableFaucet {
		err = setupFaucetKeyFiles(val)
		require.NoError(t, err)
	}
	_, err = net.WaitForHeight(1)
	require.NoError(t, err)
	t.Cleanup(func() {
		net.Cleanup()
		os.RemoveAll(baseDir)
	})
	return net
}

// DefaultConfig will initialize config for the network with custom application,
// genesis and single validator. All other parameters are inherited from cosmos-sdk/testutil/network.DefaultConfig.
func DefaultConfig() network.Config {
	return DefaultConfigWithOptions(NetworkOptions{EnableFaucet: false})
}

// DefaultConfigWithOptions will initialize config for the network with custom application,
// genesis and single validator, with optional faucet configuration.
func DefaultConfigWithOptions(options NetworkOptions) network.Config {
	app.SetConfig(false)
	cfg, err := network.DefaultConfigWithAppConfig(app.AppConfig())
	if err != nil {
		panic(err)
	}
	ports, err := freePorts(3)
	if err != nil {
		panic(err)
	}
	if cfg.APIAddress == "" {
		cfg.APIAddress = fmt.Sprintf("tcp://0.0.0.0:%s", ports[0])
	}
	if cfg.RPCAddress == "" {
		cfg.RPCAddress = fmt.Sprintf("tcp://0.0.0.0:%s", ports[1])
	}
	if cfg.GRPCAddress == "" {
		cfg.GRPCAddress = fmt.Sprintf("0.0.0.0:%s", ports[2])
	}
	cfg.BondDenom = params.DefaultBondDenom
	if options.EnableFaucet {
		if err := setupFaucetInGenesis(&cfg); err != nil {
			panic(fmt.Sprintf("failed to setup faucet in genesis: %v", err))
		}
	}
	cfg.AppConstructor = func(val network.ValidatorI) servertypes.Application {
		appOpts := NewTestAppOptions(options.EnableFaucet)
		appInstance, err := app.New(
			val.GetCtx().Logger,
			dbm.NewMemDB(),
			nil,
			true,
			appOpts,
			baseapp.SetChainID(cfg.ChainID),
		)
		if err != nil {
			panic(fmt.Sprintf("failed to create app: %v", err))
		}
		return appInstance
	}
	return cfg
}

// freePorts return the available ports based on the number of requested ports.
func freePorts(n int) ([]string, error) {
	closeFns := make([]func() error, n)
	ports := make([]string, n)
	for i := 0; i < n; i++ {
		_, port, closeFn, err := network.FreeTCPAddr()
		if err != nil {
			return nil, err
		}
		ports[i] = port
		closeFns[i] = closeFn
	}
	for _, closeFn := range closeFns {
		if err := closeFn(); err != nil {
			return nil, err
		}
	}
	return ports, nil
}

// TCPToHTTP converts a TCP address to HTTP address.
func TCPToHTTP(tcpAddr string) string {
	return strings.Replace(tcpAddr, "tcp://", "http://", 1)
}
