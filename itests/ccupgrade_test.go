package itests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/itests/kit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCCUpgrade(t *testing.T) {
	kit.QuietMiningLogs()

	for _, height := range []abi.ChainEpoch{
		-1,  // before
		162, // while sealing
		560, // after upgrade deal
	} {
		height := height // make linters happy by copying
		t.Run(fmt.Sprintf("upgrade-%d", height), func(t *testing.T) {
			runTestCCUpgrade(t, height)
		})
	}
}

func runTestCCUpgrade(t *testing.T, upgradeHeight abi.ChainEpoch) {
	t.Skip()
	ctx := context.Background()
	blockTime := 5 * time.Millisecond

	client, miner, ens := kit.EnsembleMinimal(t, kit.MockProofs(), kit.TurboUpgradeAt(upgradeHeight))
	ens.InterconnectAll().BeginMining(blockTime)

	maddr, err := miner.ActorAddress(ctx)
	if err != nil {
		t.Fatal(err)
	}

	CCUpgrade := abi.SectorNumber(kit.DefaultPresealsPerBootstrapMiner + 1)

	miner.PledgeSectors(ctx, 1, 0, nil)

	sl, err := miner.SectorsList(ctx)
	require.NoError(t, err)
	require.Len(t, sl, 1, "expected 1 sector")
	require.Equal(t, CCUpgrade, sl[0], "unexpected sector number")

	{
		si, err := client.StateSectorGetInfo(ctx, maddr, CCUpgrade, types.EmptyTSK)
		require.NoError(t, err)
		require.Less(t, 50000, int(si.Expiration))
	}

	err = miner.SectorMarkForUpgrade(ctx, sl[0])
	require.NoError(t, err)

	dh := kit.NewDealHarness(t, client, miner, miner)
	deal, res, inPath := dh.MakeOnlineDeal(ctx, kit.MakeFullDealParams{
		Rseed:                        6,
		SuspendUntilCryptoeconStable: true,
	})
	outPath := dh.PerformRetrieval(context.Background(), deal, res.Root, false)
	kit.AssertFilesEqual(t, inPath, outPath)

	status, err := miner.SectorsStatus(ctx, CCUpgrade, true)
	assert.Equal(t, 1, len(status.Deals))
}
