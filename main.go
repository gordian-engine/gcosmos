// Command gcosmos wraps the Cosmos SDK simappv2
// and adds a "gstart" subcommand to run parts of Gordian.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"

	"cosmossdk.io/core/transaction"
	"cosmossdk.io/simapp/v2"
	simdcmd "cosmossdk.io/simapp/v2/simdv2/cmd"
	sdkflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/gordian-engine/gcosmos/internal/gci"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	homeDir := simapp.DefaultNodeHome

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var rootCmd *cobra.Command
	if gci.RunCometInsteadOfGordian {
		rootCmd = simdcmd.NewCometBFTRootCmd[transaction.Tx]()
	} else {
		rootCmd = gci.NewSimdRootCmdWithGordian(
			ctx,
			// Setting the logger this way is likely to cause interleaved writes
			// with the SDK logging system, but this is fine as a quick solution.
			slog.New(slog.NewTextHandler(os.Stderr, nil)),
			homeDir,
		)
	}
	if err := Execute(ctx, rootCmd, "", homeDir); err != nil {
		fmt.Fprintln(rootCmd.OutOrStderr(), err)
		os.Exit(1)
	}
}

// Execute executes the root command of an application. It handles creating a
// server context object with the appropriate server and client objects injected
// into the underlying stdlib Context. It also handles adding core CLI flags,
// specifically the logging flags. It returns an error upon execution failure.
func Execute(ctx context.Context, rootCmd *cobra.Command, envPrefix, defaultHome string) error {
	ctx = svrcmd.CreateExecuteContext(ctx)

	rootCmd.PersistentFlags().String(sdkflags.FlagLogLevel, zerolog.InfoLevel.String(), "The logging level (trace|debug|info|warn|error|fatal|panic|disabled or '*:<level>,<key>:<level>')")
	rootCmd.PersistentFlags().String(sdkflags.FlagLogFormat, "plain", "The logging format (json|plain)")
	rootCmd.PersistentFlags().Bool(sdkflags.FlagLogNoColor, false, "Disable colored logs")
	rootCmd.PersistentFlags().StringP(sdkflags.FlagHome, "", defaultHome, "directory for config and data")
	rootCmd.PersistentFlags().Bool(server.FlagTrace, false, "print out full stack trace on errors")

	// update the global viper with the root command's configuration
	viper.SetEnvPrefix(envPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	return rootCmd.ExecuteContext(ctx)
}
