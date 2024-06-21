package main_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	simdcmd "cosmossdk.io/simapp/v2/simdv2/cmd"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/rollchains/gordian/gcosmos/internal/gci"
	"github.com/stretchr/testify/require"
)

func TestRootCmd(t *testing.T) {
	t.Parallel()

	e := NewRootCmd(t)

	e.Run("init", "defaultmoniker").NoError(t)
}

func TestRootCmd_checkGenesisValidators(t *testing.T) {
	t.Parallel()

	e := NewRootCmd(t)

	// It would be nice to be able to use deterministic keys for these tests,
	// but since the key subcommands use readers hanging off a "client context",
	// it is surprisingly difficult to intercept that value at the right spot.
	// Unfortunately, we can't simply set cmd.Input for this.
	const nVals = 4
	for i := range nVals {
		e.Run("keys", "add", fmt.Sprintf("val%d", i)).NoError(t)
	}

	e.Run("init", "testmoniker", "--chain-id", t.Name()).NoError(t)

	for i := range nVals {
		e.Run(
			"genesis", "add-genesis-account",
			fmt.Sprintf("val%d", i), "100stake",
		).NoError(t)
	}

	gentxDir := t.TempDir()

	for i := range nVals {
		vs := fmt.Sprintf("val%d", i)
		e.Run(
			"genesis", "gentx",
			"--chain-id", t.Name(),
			"--output-document", filepath.Join(gentxDir, vs+".gentx.json"),
			vs, "100stake",
		).NoError(t)
	}

	e.Run(
		"genesis", "collect-gentxs", "--gentx-dir", gentxDir,
	).NoError(t)

	res := e.Run("gstart")
	res.NoError(t)

	pubkeys := 0
	for _, line := range strings.Split(res.Stdout.String(), "\n") {
		if strings.Contains(line, "pubkey") {
			pubkeys++
		}
	}
	require.Equal(t, nVals, pubkeys)
}

func NewRootCmd(
	t *testing.T,
) CmdEnv {
	t.Helper()

	return CmdEnv{homeDir: t.TempDir()}
}

type CmdEnv struct {
	homeDir string
}

func (e CmdEnv) Run(args ...string) RunResult {
	return e.RunWithInput(nil, args...)
}

func (e CmdEnv) RunWithInput(in io.Reader, args ...string) RunResult {
	cmd := simdcmd.NewRootCmd()
	cmd.AddCommand(gci.StartGordianCommand())
	cmd.AddCommand(gci.CheckGenesisValidatorsCommand())

	// Just add the home flag directly instead of
	// relying on the comet CLI integration in the SDK.
	// Might be brittle, but should also be a little simpler.
	cmd.PersistentFlags().StringP("home", "", e.homeDir, "default test dir home, do not change")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = svrcmd.CreateExecuteContext(ctx)

	// Putting --home before the args would probably work,
	// but put --home at the end to be a little more sure
	// that it won't get ignored due to being parsed before the subcommand name.
	args = append(slices.Clone(args), "--home", e.homeDir)
	cmd.SetArgs(args)

	var res RunResult
	cmd.SetOut(&res.Stdout)
	cmd.SetErr(&res.Stderr)
	cmd.SetIn(in)

	res.Err = cmd.ExecuteContext(ctx)

	return res
}

type RunResult struct {
	Stdout, Stderr bytes.Buffer
	Err            error
}

func (r RunResult) NoError(t *testing.T) {
	t.Helper()

	require.NoErrorf(t, r.Err, "OUT: %s\n\nERR: %s", r.Stdout.String(), r.Stderr.String())
}
