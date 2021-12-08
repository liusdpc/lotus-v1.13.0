package sealing

import (
	statemachine "github.com/filecoin-project/go-statemachine"
	"github.com/ipfs/go-cid"
	"golang.org/x/xerrors"
)

func (m *Sealing) handleReplicaUpdate(ctx statemachine.Context, sector SectorInfo) error {
	if err := checkPieces(ctx.Context(), m.maddr, sector, m.Api); err != nil { // Sanity check state
		switch err.(type) {
		case *ErrApi:
			log.Errorf("handleReplicaUpdate: api error, not proceeding: %+v", err)
			return nil
		case *ErrInvalidDeals:
			log.Warnf("invalid deals in sector %d: %v", sector.SectorNumber, err)
			return ctx.Send(SectorInvalidDealIDs{Return: RetPreCommit1})
		case *ErrExpiredDeals: // Probably not much we can do here, maybe re-pack the sector?
			return ctx.Send(SectorDealsExpired{xerrors.Errorf("expired dealIDs in sector: %w", err)})
		default:
			return xerrors.Errorf("checkPieces sanity check error: %w", err)
		}
	}

	out, err := m.sealer.ReplicaUpdate(sector.sealingCtx(ctx.Context()), m.minerSector(sector.SectorType, sector.SectorNumber), sector.pieceInfos())
	if err != nil {
		// XXX error handling events and states
		panic("bad replica update")
	}
	return ctx.Send(SectorReplicaUpdate{
		Out: out,
	})
}

func (m *Sealing) handleProveReplicaUpdate1(ctx statemachine.Context, sector SectorInfo) error {
	var newSealed, newUnsealed cid.Cid
	if sector.ReplicaUpdateOut == nil {
		return xerrors.Errorf("invalid sector %d with nil ReplicaUpdate output", sector.SectorNumber)
	} else {
		newSealed, newUnsealed = sector.ReplicaUpdateOut.NewSealed, sector.ReplicaUpdateOut.NewUnsealed
	}
	if sector.CommR == nil {
		return xerrors.Errorf("invalid sector %d with nil CommR", sector.SectorNumber)
	}
	vanillaProofs, err := m.sealer.ProveReplicaUpdate1(sector.sealingCtx(ctx.Context()), m.minerSector(sector.SectorType, sector.SectorNumber), *sector.CommR, newSealed, newUnsealed)
	if err != nil {
		// XXX error handling events and states
		panic("bad prove replica update 1")
	}
	return ctx.Send(SectorProveReplicaUpdate1{
		Out: vanillaProofs,
	})
}

func (m *Sealing) handleProveReplicaUpdate2(ctx statemachine.Context, sector SectorInfo) error {

	var newSealed, newUnsealed cid.Cid
	if sector.ReplicaUpdateOut == nil {
		return xerrors.Errorf("invalid sector %d with nil ReplicaUpdate output", sector.SectorNumber)
	} else {
		newSealed, newUnsealed = sector.ReplicaUpdateOut.NewSealed, sector.ReplicaUpdateOut.NewUnsealed
	}
	if sector.CommR == nil {
		return xerrors.Errorf("invalid sector %d with nil CommR", sector.SectorNumber)
	}
	if sector.ProveReplicaUpdate1Out == nil {
		return xerrors.Errorf("invalid sector %d with nil ProveReplicaUpdate1 output", sector.SectorNumber)
	}
	proof, err := m.sealer.ProveReplicaUpdate2(sector.sealingCtx(ctx.Context()), m.minerSector(sector.SectorType, sector.SectorNumber), *sector.CommR, newSealed, newUnsealed, sector.ProveReplicaUpdate1Out)
	if err != nil {
		// XXX error handling events and states
		panic("bad prove replica update 2")
	}
	return ctx.Send(SectorProveReplicaUpdate2{
		Proof: proof,
	})
}

func (m *Sealing) handleSubmitReplicaUpdate(ctx statemachine.Context, sector SectorInfo) error {
	return nil
}
