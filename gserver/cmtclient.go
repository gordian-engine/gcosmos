package gserver

import (
	"context"
	"fmt"

	cmtcryptoed25519 "github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/libs/bytes"
	cmtclient "github.com/cometbft/cometbft/rpc/client"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	cmttypes "github.com/cometbft/cometbft/types"
	client "github.com/cosmos/cosmos-sdk/client"
	ggrpc "github.com/gordian-engine/gcosmos/gserver/internal/ggrpc"
	"github.com/spf13/cobra"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ client.CometRPC = (*Client)(nil)

type Client struct {
	cmd     *cobra.Command
	gclient ggrpc.GordianGRPCClient
}

func NewClient(
	cmd *cobra.Command,
	grpcAddress string,
	grpcInsecure bool,
) (*Client, error) {
	var dialOpts []grpc.DialOption
	if grpcInsecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	cc, err := grpc.NewClient(grpcAddress, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC: %w", err)
	}

	return &Client{
		cmd:     cmd,
		gclient: ggrpc.NewGordianGRPCClient(cc),
	}, nil
}

func (c *Client) ABCIInfo(ctx context.Context) (*coretypes.ResultABCIInfo, error) {
	panic(fmt.Errorf("not implemented"))
}

func (c *Client) ABCIQuery(ctx context.Context, path string, data bytes.HexBytes) (*coretypes.ResultABCIQuery, error) {
	panic(fmt.Errorf("not implemented"))
}

func (c *Client) ABCIQueryWithOptions(ctx context.Context, path string, data bytes.HexBytes, opts cmtclient.ABCIQueryOptions) (*coretypes.ResultABCIQuery, error) {
	panic(fmt.Errorf("not implemented"))
}

func (c *Client) BroadcastTxAsync(ctx context.Context, tx cmttypes.Tx) (*coretypes.ResultBroadcastTx, error) {
	// TODO implement
	return c.BroadcastTxSync(ctx, tx)
}

func (c *Client) BroadcastTxCommit(ctx context.Context, tx cmttypes.Tx) (*coretypes.ResultBroadcastTxCommit, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *Client) BroadcastTxSync(ctx context.Context, tx cmttypes.Tx) (*coretypes.ResultBroadcastTx, error) {
	clientCtx, err := client.GetClientTxContext(c.cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get client tx context: %w", err)
	}

	sdkTx, err := clientCtx.TxConfig.TxDecoder()(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tx: %w", err)
	}

	txJson, err := clientCtx.TxConfig.TxJSONEncoder()(sdkTx)
	if err != nil {
		return nil, fmt.Errorf("failed to encode tx: %w", err)
	}

	res, err := c.gclient.SimulateTransaction(ctx, &ggrpc.SubmitSimulationTransactionRequest{
		Tx: txJson,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to simulate transaction: %w", err)
	}
	if res.Error != "" {
		return nil, fmt.Errorf("failed to simulate transaction: %s", res.Error)
	}

	res, err = c.gclient.SubmitTransaction(ctx, &ggrpc.SubmitTransactionRequest{
		Tx: txJson,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to submit transaction: %w", err)
	}
	if res.Error != "" {
		return nil, fmt.Errorf("failed to submit transaction: %s", res.Error)
	}

	return &coretypes.ResultBroadcastTx{
		Code:      res.Code,
		Data:      res.Data,
		Log:       res.Log,
		Hash:      tx.Hash(),
		Codespace: res.Codespace,
	}, nil
}

func (c *Client) Validators(ctx context.Context, height *int64, page, perPage *int) (*coretypes.ResultValidators, error) {
	valsRes, err := c.gclient.GetValidators(ctx, &ggrpc.GetValidatorsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get validators: %w", err)
	}

	vals := valsRes.Validators

	res := &coretypes.ResultValidators{
		BlockHeight: 0, // TODO
		Validators:  make([]*cmttypes.Validator, len(vals)),
		Count:       len(vals), // no pagination yet
		Total:       len(vals),
	}

	for i, val := range vals {
		res.Validators[i] = &cmttypes.Validator{
			// Address: // TODO derive from pubkey
			PubKey:      cmtcryptoed25519.PubKey(val.GetEncodedPubKey()), // TODO may need to decode based on method name
			VotingPower: int64(val.Power),
		}
	}

	return res, nil
}

func (c *Client) Status(ctx context.Context) (*coretypes.ResultStatus, error) {
	gRes, err := c.gclient.GetBlocksWatermark(ctx, &ggrpc.CurrentBlockRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to get blocks watermark: %w", err)
	}

	// TODO fill out remaining fields
	return &coretypes.ResultStatus{
		SyncInfo: coretypes.SyncInfo{
			LatestBlockHeight: int64(gRes.CommittingHeight) - 1,
		},
	}, nil
}

func (c *Client) Block(ctx context.Context, height *int64) (*coretypes.ResultBlock, error) {
	panic(fmt.Errorf("not implemented"))
}

func (c *Client) BlockByHash(ctx context.Context, hash []byte) (*coretypes.ResultBlock, error) {
	panic(fmt.Errorf("not implemented"))
}

func (c *Client) BlockResults(ctx context.Context, height *int64) (*coretypes.ResultBlockResults, error) {
	panic(fmt.Errorf("not implemented"))
}

func (c *Client) BlockchainInfo(ctx context.Context, minHeight, maxHeight int64) (*coretypes.ResultBlockchainInfo, error) {
	panic(fmt.Errorf("not implemented"))
}

func (c *Client) Commit(ctx context.Context, height *int64) (*coretypes.ResultCommit, error) {
	panic(fmt.Errorf("not implemented"))
}

func (c *Client) Tx(ctx context.Context, hash []byte, prove bool) (*coretypes.ResultTx, error) {
	panic(fmt.Errorf("not implemented"))
}

func (c *Client) TxSearch(
	ctx context.Context,
	query string,
	prove bool,
	page, perPage *int,
	orderBy string,
) (*coretypes.ResultTxSearch, error) {
	panic(fmt.Errorf("not implemented"))
}

func (c *Client) BlockSearch(
	ctx context.Context,
	query string,
	page, perPage *int,
	orderBy string,
) (*coretypes.ResultBlockSearch, error) {
	panic(fmt.Errorf("not implemented"))
}
