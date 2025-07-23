package cmd

import (
	"fmt"

	cmtcfg "github.com/cometbft/cometbft/config"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	appparams "github.com/sourcenetwork/sourcehub/app/params"
)

// CustomAppConfig extends the default Cosmos SDK app config
type CustomAppConfig struct {
	serverconfig.Config `mapstructure:",squash"`

	Faucet FaucetConfig `mapstructure:"faucet"`
}

// FaucetConfig defines the configuration for the faucet service
type FaucetConfig struct {
	EnableFaucet bool `mapstructure:"enable_faucet"`
}

// initCometBFTConfig helps to override default CometBFT Config values.
// return cmtcfg.DefaultConfig if no custom configuration is required for the application.
func initCometBFTConfig() *cmtcfg.Config {
	cfg := cmtcfg.DefaultConfig()

	// these values put a higher strain on node memory
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	return cfg
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	// Optionally allow the chain developer to overwrite the SDK's default server config.
	srvCfg := serverconfig.DefaultConfig()
	srvCfg.MinGasPrices = fmt.Sprintf(
		"%s%s,%s%s",
		appparams.DefaultMinGasPrice,
		appparams.MicroOpenDenom,
		appparams.DefaultMinGasPrice,
		appparams.MicroCreditDenom,
	)
	// The SDK's default minimum gas price is set to "" (empty value) inside
	// app.toml. If left empty by validators, the node will halt on startup.
	// However, the chain developer can set a default app.toml value for their
	// validators here.
	//
	// In summary:
	// - if you leave srvCfg.MinGasPrices = "", all validators MUST tweak their
	//   own app.toml config,
	// - if you set srvCfg.MinGasPrices non-empty, validators CAN tweak their
	//   own app.toml to override, or use this default value.
	//
	// In tests, we set the min gas prices to 0.
	// srvCfg.MinGasPrices = "0stake"
	// srvCfg.BaseConfig.IAVLDisableFastNode = true // disable fastnode by default

	customAppConfig := CustomAppConfig{
		Config: *srvCfg,
		Faucet: FaucetConfig{
			EnableFaucet: false,
		},
	}

	customAppTemplate := serverconfig.DefaultConfigTemplate + `
###############################################################################
###                           Faucet Configuration                          ###
###############################################################################

[faucet]

# Enable defines if the faucet service should be enabled.
enable_faucet = {{ .Faucet.EnableFaucet }}
`

	return customAppTemplate, customAppConfig
}
