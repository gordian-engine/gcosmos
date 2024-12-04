package gci

import (
	"fmt"
	"io"

	"cosmossdk.io/client/v2/offchain"
	coreserver "cosmossdk.io/core/server"
	"cosmossdk.io/core/transaction"
	clog "cosmossdk.io/log"
	runtimev2 "cosmossdk.io/runtime/v2"
	serverv2 "cosmossdk.io/server/v2"
	grpcserver "cosmossdk.io/server/v2/api/grpc"
	"cosmossdk.io/server/v2/api/rest"
	"cosmossdk.io/server/v2/api/telemetry"
	"cosmossdk.io/server/v2/appmanager"
	serverstore "cosmossdk.io/server/v2/store"
	"cosmossdk.io/store/v2"
	confixcmd "cosmossdk.io/tools/confix/cmd"
	"github.com/cosmos/cosmos-sdk/client"
	clientdebug "github.com/cosmos/cosmos-sdk/client/debug"
	"github.com/cosmos/cosmos-sdk/client/keys"
	clientrpc "github.com/cosmos/cosmos-sdk/client/rpc"
	sdktelemetry "github.com/cosmos/cosmos-sdk/telemetry"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	clientcli "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutilv1cli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	genutilv2 "github.com/cosmos/cosmos-sdk/x/genutil/v2"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/v2/cli"
	"github.com/gordian-engine/gcosmos/gcapp"
	"github.com/gordian-engine/gcosmos/gserver"
	"github.com/spf13/cobra"
)

// addExtraCommands adds the standard commands that are supposed to always be available.
func addExtraCommands(
	cmd *cobra.Command,
	mm *runtimev2.MM[transaction.Tx],
) {
	queryCmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}
	queryCmd.AddCommand(
		clientrpc.QueryEventForTxCmd(),
		clientcli.QueryTxsByEventsCmd(),
		clientcli.QueryTxCmd(),
	)

	txCmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         false,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	txCmd.AddCommand(
		clientcli.GetSignCommand(),
		clientcli.GetSignBatchCommand(),
		clientcli.GetMultiSignCommand(),
		clientcli.GetMultiSignBatchCmd(),
		clientcli.GetValidateSignaturesCommand(),
		clientcli.GetBroadcastCommand(),
		clientcli.GetEncodeCommand(),
		clientcli.GetDecodeCommand(),
		clientcli.GetSimulateCmd(),
	)

	cmd.AddCommand(
		genutilv1cli.InitCmd(mm),
		xxgenesisCommand(mm, nil /*deps.SimApp*/), // We need to ensure our app satisfies the interface.
		// There used to be a call to NewTestnetCmd here, but I don't think we will keep that.
		clientdebug.Cmd(),
		confixcmd.ConfigCommand(),
		queryCmd,
		txCmd,
		keys.Commands(),
		offchain.OffChain(),
	)
}

func initRootCommandClientOnly(
	cmd *cobra.Command,
	logger clog.Logger,
	component *gserver.Component,
) (serverv2.ConfigWriter, error) {
	cfg := sdktypes.GetConfig()
	cfg.Seal()

	// Extracted from SDK simdv2/cmd/config.go: initServerConfig
	initServerCfg := serverv2.DefaultServerConfig()
	initServerCfg.MinGasPrices = "0stake"

	return serverv2.AddCommands[transaction.Tx](
		cmd,
		logger,
		io.NopCloser(nil),
		nil,           // Global app config, unset for client-only.
		initServerCfg, // Global server config, is just the initial config for client-only.
		component,     // simd had a derfault comet server at this point.
		// Then a bunch of default server values. Unclear if these are actually necessary.
		&grpcserver.Server[transaction.Tx]{},
		&serverstore.Server[transaction.Tx]{},
		&telemetry.Server[transaction.Tx]{},
		&rest.Server[transaction.Tx]{},
	)
}

type fullRootCommandConfig struct {
	App *gcapp.GCApp

	ConsensusComponent *gserver.Component

	// Should come from the configMap return value from factory.ParseCommand.
	GlobalConfig coreserver.ConfigMap

	// Created while making the command.
	TxConfig client.Context

	// This previously came from SimApp.
	// Not sure where we will get it now yet.
	RootStore store.RootStore

	// Would come from simApp.App.AppManager.
	AppManager appmanager.AppManager[transaction.Tx]
}

func initRootCommandFull(
	cmd *cobra.Command,
	logger clog.Logger,
	fullCfg fullRootCommandConfig,
	// simdv2 uses "CommandDependencies"
) (serverv2.ConfigWriter, error) {
	cfg := sdktypes.GetConfig()
	cfg.Seal()

	storeComponent, err := serverstore.New[transaction.Tx](fullCfg.RootStore, fullCfg.GlobalConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create server store: %w", err)
	}
	restServer, err := rest.New[transaction.Tx](logger, fullCfg.AppManager, fullCfg.GlobalConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create rest server: %w", err)
	}

	telemetryServer, err := telemetry.New[transaction.Tx](fullCfg.GlobalConfig, logger, sdktelemetry.EnableTelemetry)
	if err != nil {
		return nil, fmt.Errorf("failed to create telemetry server: %w", err)
	}

	// TODO: allow gRPC server
	// grpcServer, err := grpcserver.New[transaction.Tx](
	// 	logger,
	// 	simApp.InterfaceRegistry(),
	// 	simApp.QueryHandlers(),
	// 	simApp.Query,
	// 	fullCfg.GlobalConfig,
	// 	grpcserver.WithExtraGRPCHandlers[transaction.Tx](
	// 		deps.ConsensusServer.GRPCServiceRegistrar(
	// 			fullCfg.ClientContext,
	// 			fullCfg.GlobalConfig,
	// 		),
	// 	),
	// )
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create gRPC server: %w", err)
	// }

	// Extracted from SDK simdv2/cmd/config.go: initServerConfig
	initServerCfg := serverv2.DefaultServerConfig()
	initServerCfg.MinGasPrices = "0stake"

	var closer io.Closer = fullCfg.App // TODO: fallback in nil case

	return serverv2.AddCommands[transaction.Tx](
		cmd,
		logger,
		closer,
		fullCfg.GlobalConfig,
		initServerCfg,
		fullCfg.ConsensusComponent,
		// grpcServer,
		storeComponent,
		telemetryServer,
		restServer,
	)
}

func xxgenesisCommand[T transaction.Tx](
	mm *runtimev2.MM[T],
	app interface {
		// TODO: this is an inline equivalent to x/genutil/v2/cli.ExportableApp;
		// LoadHeight is part of runtime.App but ExportAppStateAndValidators
		// is directly implemented on simapp.
		ExportAppStateAndValidators([]string) (genutilv2.ExportedApp, error)
		LoadHeight(uint64) error
	},
) *cobra.Command {
	var genTxValidator func([]transaction.Msg) error
	if mm != nil {
		genTxValidator = mm.Modules()[genutiltypes.ModuleName].(genutil.AppModule).GenTxValidator()
	}

	return genutilcli.Commands(genTxValidator, mm, app)
}
