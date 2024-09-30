package tmstoretest

import (
	"context"
	"testing"

	"github.com/rollchains/gordian/tm/tmconsensus"
	"github.com/rollchains/gordian/tm/tmconsensus/tmconsensustest"
	"github.com/rollchains/gordian/tm/tmstore"
	"github.com/stretchr/testify/require"
)

type HeaderStoreFactory func(cleanup func(func())) (tmstore.HeaderStore, error)

func TestHeaderStoreCompliance(t *testing.T, f HeaderStoreFactory) {
	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s, err := f(t.Cleanup)
		require.NoError(t, err)

		fx := tmconsensustest.NewStandardFixture(4)

		ph1 := fx.NextProposedHeader([]byte("app_data_1"), 0)
		ph1.Header.PrevAppStateHash = []byte("initial_app_state") // TODO: this should be automatically set in the fixture.

		// This tends to be nil from the store.
		// Ensure it is nil here too.
		require.Empty(t, ph1.Header.PrevCommitProof.Proofs)
		ph1.Header.PrevCommitProof.Proofs = nil

		fx.RecalculateHash(&ph1.Header)
		fx.SignProposal(ctx, &ph1, 0)

		voteMap := map[string][]int{
			string(ph1.Header.Hash): {0, 1, 2},
			"":                      {3},
		}
		precommitProofs1 := fx.PrecommitProofMap(ctx, 1, 0, voteMap)
		fx.CommitBlock(ph1.Header, []byte("app_state_height_1"), 0, precommitProofs1)

		ph2 := fx.NextProposedHeader([]byte("app_data_2"), 0)
		ph2.Round = 1

		ch := tmconsensus.CommittedHeader{
			Header: ph1.Header,
			Proof:  ph2.Header.PrevCommitProof,
		}

		require.NoError(t, s.SaveHeader(ctx, ch))

		got, err := s.LoadHeader(ctx, 1)
		require.NoError(t, err)

		require.Equal(t, ch, got)

		voteMap = map[string][]int{
			string(ph2.Header.Hash): {0, 1, 3},
			"":                      {2},
		}
		precommitProofs2 := fx.PrecommitProofMap(ctx, 2, 1, voteMap)
		fx.CommitBlock(ph2.Header, []byte("app_state_height_2"), 1, precommitProofs2)

		ph3 := fx.NextProposedHeader([]byte("app_data_3"), 0)

		ch = tmconsensus.CommittedHeader{
			Header: ph2.Header,
			Proof:  ph3.Header.PrevCommitProof,
		}

		require.NoError(t, s.SaveHeader(ctx, ch))

		got, err = s.LoadHeader(ctx, 2)
		require.NoError(t, err)

		require.Equal(t, ch, got)
	})
}
