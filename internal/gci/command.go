package gci

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
	"cosmossdk.io/client/v2/autocli"
	coreserver "cosmossdk.io/core/server"
	"cosmossdk.io/core/transaction"
	"cosmossdk.io/depinject"
	clog "cosmossdk.io/log"
	cslog "cosmossdk.io/log/slog"
	"cosmossdk.io/runtime/v2"
	serverv2 "cosmossdk.io/server/v2"
	"github.com/cosmos/cosmos-sdk/client"
	sdkflags "github.com/cosmos/cosmos-sdk/client/flags"
	nodeservice "github.com/cosmos/cosmos-sdk/client/grpc/node"
	"github.com/gordian-engine/gcosmos/gcapp"
	"github.com/gordian-engine/gcosmos/gccodec"
	"github.com/gordian-engine/gcosmos/gserver"
	"github.com/spf13/cobra"
)

func init() {
	sdkflags.CliNodeFlagOpts = &sdkflags.CmdNodeFlagOpts{
		QueryOpts: &sdkflags.NodeFlagOpts{DefaultGRPC: "localhost:9090"},
		TxOpts:    &sdkflags.NodeFlagOpts{DefaultGRPC: "localhost:9092"},
	}
}

// TODO: this should update to return an error instead of panicking.
func NewGcosmosCommand(
	lifeCtx context.Context,
	log *slog.Logger,
	homeDir string,
	args []string,
) *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "gcosmos",
	}

	// NOTE: this is not supposed to be a full component.
	// This is used to set up the initial command structure,
	// and we can be assured that the returned gserver.Component
	// will not have its Start method called.
	//
	// See NewWithConfigOptions in server/v2/cometbft/server.go.
	// "It is *not* a fully functional server (since it has been created without dependencies)
	// The returned server should only be used to get and set configuration."
	component, err := gserver.NewComponent(lifeCtx, log, homeDir, nil, nil, gserver.Config{})
	if err != nil {
		panic(fmt.Errorf("failed to create gserver component: %w", err))
	}

	addExtraCommands(rootCmd, nil)

	configWriter, err := initRootCommandClientOnly(
		rootCmd,
		cslog.NewCustomLogger(log),
		component,
	)
	if err != nil {
		panic(fmt.Errorf("failed to initialize client-only root command: %w", err))
	}

	factory, err := serverv2.NewCommandFactory(
		serverv2.WithConfigWriter(configWriter),
		serverv2.WithStdDefaultHomeDir(homeDir),
		serverv2.WithLoggerFactory(func(coreserver.ConfigMap, io.Writer) (clog.Logger, error) {
			return cslog.NewCustomLogger(log), nil
		}),
	)
	if err != nil {
		panic(fmt.Errorf("failed to build server command factory: %w", err))
	}

	rootCmd.PersistentFlags().String("chain-id", "", "TODO: don't set this in internal/gci/command.go")
	rootCmd.PersistentFlags().String("from", "", "TODO: don't set this in internal/gci/command.go")
	rootCmd.PersistentFlags().Bool("generate-only", false, "TODO: don't set this in internal/gci/command.go")

	rootCmd.PersistentFlags().String("grpc-addr", "", "TODO: don't set this in internal/gci/command.go")
	rootCmd.PersistentFlags().String("grpc-insecure", "", "TODO: don't set this in internal/gci/command.go")

	subCommand, configMap, logger, err := factory.ParseCommand(rootCmd, args)
	if err != nil {
		panic(fmt.Errorf("failed to parse command [args=%#v]: %w", args, err))
	}

	var (
		autoCliOpts     autocli.AppOptions
		moduleManager   *runtime.MM[transaction.Tx]
		clientCtx       client.Context
		depinjectConfig = depinject.Configs(
			depinject.Supply(logger, runtime.GlobalConfig(configMap)),
			depinject.Provide(gcapp.ProvideClientContext),
		)
	)
	clientCtx = clientCtx.WithHomeDir(homeDir).WithViper("")
	clientCtx.Viper.SetDefault("home", homeDir)

	var gcApp *gcapp.GCApp
	if serverv2.IsAppRequired(subCommand) {
		// server construction

		gcApp, err = gcapp.NewGCApp(depinjectConfig, &autoCliOpts, &moduleManager, &clientCtx)
		if err != nil {
			panic(fmt.Errorf("failed to construct new gcapp: %w", err))
		}

		// Overwrite the component.
		component, err = gserver.NewComponent(
			lifeCtx,
			log,
			homeDir,
			gccodec.NewTxDecoder(clientCtx.TxConfig),
			clientCtx.Codec,
			gserver.Config{
				RootStore:  gcApp.Store(),
				AppManager: gcApp.AppManager,
				ConfigMap:  configMap,
			},
		)
		if err != nil {
			panic(fmt.Errorf("failed to build gserver component: %w", err))
		}
	} else {
		// client construction
		if err = depinject.Inject(
			depinject.Configs(
				gcapp.AppConfig(),
				depinjectConfig,
			),
			&autoCliOpts, &moduleManager, &clientCtx,
		); err != nil {
			panic(fmt.Errorf("failed to depinject client: %w", err))
		}
	}

	// We have to overwrite the previously use rootCmd here,
	// because the old value is now associated with potentially invalid values,
	// and we have updated depinject values that correlate to the command being invoked.
	rootCmd = &cobra.Command{
		Use:               "gcosmos",
		PersistentPreRunE: gcapp.RootCommandPersistentPreRun(clientCtx),
	}
	rootCmd.SetContext(lifeCtx)
	factory.EnhanceRootCommand(rootCmd)

	addExtraCommands(rootCmd, moduleManager)

	if _, err = initRootCommandFull(
		rootCmd,
		cslog.NewCustomLogger(log),
		fullRootCommandConfig{
			App:                gcApp,
			ConsensusComponent: component,
		},
	); err != nil {
		panic(fmt.Errorf("failed to initialize full root command: %w", err))
	}

	nodeCmds := nodeservice.NewNodeCommands()
	autoCliOpts.ModuleOptions = make(map[string]*autocliv1.ModuleOptions)
	autoCliOpts.ModuleOptions[nodeCmds.Name()] = nodeCmds.AutoCLIOptions()
	if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
		panic(fmt.Errorf("failed to enhance root command: %w", err))
	}

	shimGordianClient(rootCmd)

	return rootCmd
}

func shimGordianClient(cmd *cobra.Command) {
	origPersistentPreRunE := cmd.PersistentPreRunE

	// Override TM RPC client on clientCtx to use gordian's gRPC client.
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := origPersistentPreRunE(cmd, args); err != nil {
			return err
		}

		grpcAddress, _ := cmd.Flags().GetString(sdkflags.FlagGRPCTx)
		if grpcAddress == "" {
			return nil
		}

		grpcInsecure, _ := cmd.Flags().GetBool(sdkflags.FlagGRPCInsecure)

		clientShim, err := gserver.NewClient(cmd, grpcAddress, grpcInsecure)
		if err != nil {
			return fmt.Errorf("failed to create gRPC client: %w", err)
		}

		clientCtx := client.GetClientContextFromCmd(cmd)
		clientCtx = clientCtx.WithClient(clientShim)

		return client.SetCmdClientContext(cmd, clientCtx)
	}
}
