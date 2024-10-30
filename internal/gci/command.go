package gci

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"cosmossdk.io/core/transaction"
	serverv2 "cosmossdk.io/server/v2"
	simdcmd "cosmossdk.io/simapp/v2/simdv2/cmd"
	"github.com/cosmos/cosmos-sdk/client"
	sdkflags "github.com/cosmos/cosmos-sdk/client/flags"
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

// NewSimdRootCmdWithGordian calls a simdcmd function we have added
// in order to get simd start to use Gordian instead of Comet.
func NewSimdRootCmdWithGordian(lifeCtx context.Context, log *slog.Logger, homeDir string) *cobra.Command {
	cmd := simdcmd.NewRootCmdWithConsensusComponent(func(cc client.Context) serverv2.ServerComponent[transaction.Tx] {
		codec := gccodec.NewTxDecoder(cc.TxConfig)

		dataDir := filepath.Join(homeDir, "data")
		c, err := gserver.NewComponent(lifeCtx, log, dataDir, codec, cc.Codec)
		if err != nil {
			panic(err)
		}
		return c
	})

	rootPersistentPreRunE := cmd.PersistentPreRunE

	// Override TM RPC client on clientCtx to use gordian's gRPC client.
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := rootPersistentPreRunE(cmd, args); err != nil {
			return err
		}

		grpcAddress, _ := cmd.Flags().GetString(sdkflags.FlagGRPCConsensus)
		if grpcAddress == "" {
			return nil
		}

		grpcInsecure, _ := cmd.Flags().GetBool(sdkflags.FlagGRPCInsecureConsensus)

		clientShim, err := gserver.NewClient(cmd, grpcAddress, grpcInsecure)
		if err != nil {
			return fmt.Errorf("failed to create gordian gRPC client: %w", err)
		}

		clientCtx := client.GetClientContextFromCmd(cmd)
		clientCtx = clientCtx.WithClient(clientShim)

		return client.SetCmdClientContext(cmd, clientCtx)
	}

	return cmd
}
