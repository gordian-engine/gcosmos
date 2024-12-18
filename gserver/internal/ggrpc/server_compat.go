package ggrpc

import (
	"context"
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/event"
	coreserver "cosmossdk.io/core/server"
	banktypes "cosmossdk.io/x/bank/types"
)

// This file contains methods on GordianGRPC that satisfy
// the server interfaces to meet SDK CLI expectations.

// SubmitTransaction implements GordianGRPCServer.
func (g *GordianGRPC) SubmitTransaction(ctx context.Context, req *SubmitTransactionRequest) (*TxResultResponse, error) {
	b := req.Tx
	tx, err := g.txc.DecodeJSON(b)
	if err != nil {
		return &TxResultResponse{
			Error: fmt.Sprintf("failed to decode transaction json: %v", err),
		}, nil
	}

	res, err := g.am.ValidateTx(ctx, tx)
	if err != nil {
		// ValidateTx should only return an error at this level,
		// if it failed to get state from the store.
		g.log.Warn("Error attempting to validate transaction", "route", "submit_tx", "err", err)
		return nil, fmt.Errorf("failed to validate transaction: %w", err)
	}

	if res.Error != nil {
		// This is fine from the server's perspective, no need to log.
		return &TxResultResponse{
			Error: fmt.Sprintf("failed to validate transaction: %v", res.Error),
		}, nil
	}

	// If it passed basic validation, then we can attempt to add it to the buffer.
	if err := g.txBuf.AddTx(ctx, tx); err != nil {
		// We could potentially check if it is a TxInvalidError here
		// and adjust the status code,
		// but since this is a debug endpoint, we'll ignore the type.
		return nil, fmt.Errorf("failed to add transaction to buffer: %w", err)
	}

	return getGordianResponseFromSDKResult(res), nil
}

// SimulateTransaction implements GordianGRPCServer.
func (g *GordianGRPC) SimulateTransaction(ctx context.Context, req *SubmitSimulationTransactionRequest) (*TxResultResponse, error) {
	b := req.Tx
	tx, err := g.txc.DecodeJSON(b)
	if err != nil {
		return &TxResultResponse{
			Error: fmt.Sprintf("failed to decode transaction json: %v", err),
		}, nil
	}

	res, _, err := g.am.Simulate(ctx, tx)
	if err != nil {
		// Simulate should only return an error at this level,
		// if it failed to get state from the store.
		g.log.Warn("Error attempting to simulate transaction", "route", "simulate_tx", "err", err)
		return nil, fmt.Errorf("failed to simulate transaction: %w", err)
	}

	if res.Error != nil {
		// This is fine from the server's perspective, no need to log.
		return &TxResultResponse{
			Error: fmt.Sprintf("failed to simulate transaction: %v", res.Error),
		}, nil
	}

	return getGordianResponseFromSDKResult(res), nil
}

// PendingTransactions implements GordianGRPCServer.
func (g *GordianGRPC) PendingTransactions(ctx context.Context, req *PendingTransactionsRequest) (*PendingTransactionsResponse, error) {
	txs := g.txBuf.Buffered(ctx, nil)

	encodedTxs := make([][]byte, len(txs))
	for i, tx := range txs {
		b, err := json.Marshal(tx)
		if err != nil {
			return nil, fmt.Errorf("failed to encode transaction: %w", err)
		}
		encodedTxs[i] = json.RawMessage(b)
	}

	return &PendingTransactionsResponse{
		Txs: encodedTxs,
	}, nil
}

// QueryAccountBalance implements GordianGRPCServer.
func (g *GordianGRPC) QueryAccountBalance(ctx context.Context, req *QueryAccountBalanceRequest) (*QueryAccountBalanceResponse, error) {
	if req.Address == "" {
		return nil, fmt.Errorf("address field is required")
	}

	denom := "stake"
	if req.Denom != "" {
		denom = req.Denom
	}

	msg, err := g.am.Query(ctx, 0, &banktypes.QueryBalanceRequest{
		Address: req.Address,
		Denom:   denom,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query account balance: %w", err)
	}

	b, err := g.cdc.MarshalJSON(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode response: %w", err)
	}

	var val QueryAccountBalanceResponse
	if err = g.cdc.UnmarshalJSON(b, &val); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &val, nil
}

func (g *GordianGRPC) QueryTx(ctx context.Context, req *SDKQueryTxRequest) (*SDKQueryTxResponse, error) {
	height, txBytes, txResult, err := g.txs.LoadTxByHash(ctx, req.TxHash)
	if err != nil {
		// TODO: check special error values to indicate internal error vs bad request.
		panic(err)
	}

	return &SDKQueryTxResponse{
		TxHash: req.TxHash,

		Height: int64(height),

		// TODO: what do we use for the Index field?

		Result: getGordianResponseFromSDKResult(txResult),

		TxBytes: txBytes,
	}, nil
}

// getGordianResponseFromSDKResult converts an app manager TxResult to the gRPC proto result.
func getGordianResponseFromSDKResult(res coreserver.TxResult) *TxResultResponse {
	events, err := convertEvent(res.Events)
	if err != nil {
		return &TxResultResponse{
			Error: fmt.Sprintf("failed to extract result events: %v", err),
		}
	}

	resp := &TxResultResponse{
		Events:    events,
		GasWanted: res.GasWanted,
		GasUsed:   res.GasUsed,
	}
	if res.Error != nil {
		resp.Error = res.Error.Error()
	}
	return resp
}

// convertEvent converts from the cosmos-sdk core event type to the gRPC proto event.
func convertEvent(e []event.Event) ([]*Event, error) {
	events := make([]*Event, len(e))
	for i, ev := range e {
		evAttrs, err := ev.Attributes()
		if err != nil {
			return nil, err
		}

		attr := make([]*Attribute, len(evAttrs))
		for j, a := range evAttrs {
			attr[j] = &Attribute{
				Key:   a.Key,
				Value: a.Value,
			}
		}

		events[i] = &Event{
			Type:       ev.Type,
			Attributes: attr,
		}
	}
	return events, nil
}
