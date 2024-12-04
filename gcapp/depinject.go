package gcapp

import (
	"os"
	"strings"

	"cosmossdk.io/core/address"
	"cosmossdk.io/core/registry"
	"cosmossdk.io/runtime/v2"
	serverv2 "cosmossdk.io/server/v2"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	clientconfig "github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtxconfig "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/spf13/cobra"

	// Needed for depinject to work.
	_ "cosmossdk.io/x/authz"
	_ "cosmossdk.io/x/authz/module"
	_ "cosmossdk.io/x/bank"
	_ "cosmossdk.io/x/bank/v2"
	_ "cosmossdk.io/x/circuit"
	_ "cosmossdk.io/x/consensus"
	_ "cosmossdk.io/x/distribution"
	_ "cosmossdk.io/x/epochs"
	_ "cosmossdk.io/x/evidence"
	_ "cosmossdk.io/x/feegrant"
	_ "cosmossdk.io/x/feegrant/module"
	_ "cosmossdk.io/x/group"
	_ "cosmossdk.io/x/group/module"
	_ "cosmossdk.io/x/mint"
	_ "cosmossdk.io/x/nft"
	_ "cosmossdk.io/x/nft/module"
	_ "cosmossdk.io/x/protocolpool"
	_ "cosmossdk.io/x/slashing"
	_ "cosmossdk.io/x/staking"
	_ "github.com/cosmos/cosmos-sdk/x/auth"
	_ "github.com/cosmos/cosmos-sdk/x/auth/vesting"
)

// ProvideClientContext is a copy-paste of simapp's ProvideClientContext.
func ProvideClientContext(
	configMap runtime.GlobalConfig,
	appCodec codec.Codec,
	interfaceRegistry codectypes.InterfaceRegistry,
	txConfigOpts tx.ConfigOptions,
	legacyAmino registry.AminoRegistrar,
	addressCodec address.Codec,
	validatorAddressCodec address.ValidatorAddressCodec,
	consensusAddressCodec address.ConsensusAddressCodec,
) client.Context {
	var err error
	amino, ok := legacyAmino.(*codec.LegacyAmino)
	if !ok {
		panic("registry.AminoRegistrar must be an *codec.LegacyAmino instance for legacy ClientContext")
	}
	homeDir, ok := configMap[serverv2.FlagHome].(string)
	if !ok {
		panic("server.ConfigMap must contain a string value for serverv2.FlagHome")
	}

	clientCtx := client.Context{}.
		WithCodec(appCodec).
		WithInterfaceRegistry(interfaceRegistry).
		WithLegacyAmino(amino).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithAddressCodec(addressCodec).
		WithValidatorAddressCodec(validatorAddressCodec).
		WithConsensusAddressCodec(consensusAddressCodec).
		WithHomeDir(homeDir).
		WithViper("") // uses by default the binary name as prefix

	// Read the config to overwrite the default values with the values from the config file
	customClientTemplate, customClientConfig := initClientConfig()
	clientCtx, err = config.CreateClientConfig(clientCtx, customClientTemplate, customClientConfig)
	if err != nil {
		panic(err)
	}

	// textual is enabled by default, we need to re-create the tx config grpc instead of bank keeper.
	txConfigOpts.TextualCoinMetadataQueryFn = authtxconfig.NewGRPCCoinMetadataQueryFn(clientCtx)
	txConfig, err := tx.NewTxConfigWithOptions(clientCtx.Codec, txConfigOpts)
	if err != nil {
		panic(err)
	}
	clientCtx = clientCtx.WithTxConfig(txConfig)

	return clientCtx
}

// initClientConfig is a copy-paste of the simapp function of the same name.
func initClientConfig() (string, interface{}) {
	type GasConfig struct {
		GasAdjustment float64 `mapstructure:"gas-adjustment"`
	}

	type CustomClientConfig struct {
		clientconfig.Config `mapstructure:",squash"`

		GasConfig GasConfig `mapstructure:"gas"`
	}

	// Optionally allow the chain developer to overwrite the SDK's default client config.
	clientCfg := clientconfig.DefaultConfig()

	// The SDK's default keyring backend is set to "os".
	// This is more secure than "test" and is the recommended value.
	//
	// In simapp, we set the default keyring backend to test, as SimApp is meant
	// to be an example and testing application.
	clientCfg.KeyringBackend = keyring.BackendTest

	// Now we set the custom config default values.
	customClientConfig := CustomClientConfig{
		Config: *clientCfg,
		GasConfig: GasConfig{
			GasAdjustment: 1.5,
		},
	}

	// The default SDK app template is defined in serverconfig.DefaultConfigTemplate.
	// We append the custom config template to the default one.
	// And we set the default config to the custom app template.
	customClientConfigTemplate := clientconfig.DefaultClientConfigTemplate + strings.TrimSpace(`
# This is default the gas adjustment factor used in tx commands.
# It can be overwritten by the --gas-adjustment flag in each tx command.
gas-adjustment = {{ .GasConfig.GasAdjustment }}
`)

	return customClientConfigTemplate, customClientConfig
}

// RootCommandPersistentPreRun is a copy-paste of simapp's functin with the same name.
func RootCommandPersistentPreRun(clientCtx client.Context) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// set the default command outputs
		cmd.SetOut(cmd.OutOrStdout())
		cmd.SetErr(cmd.ErrOrStderr())

		clientCtx = clientCtx.WithCmdContext(cmd.Context())
		clientCtx, err := client.ReadPersistentCommandFlags(clientCtx, cmd.Flags())
		if err != nil {
			return err
		}

		customClientTemplate, customClientConfig := initClientConfig()
		clientCtx, err = config.CreateClientConfig(clientCtx, customClientTemplate, customClientConfig)
		if err != nil {
			return err
		}

		if err = client.SetCmdClientContextHandler(clientCtx, cmd); err != nil {
			return err
		}

		return nil
	}
}
