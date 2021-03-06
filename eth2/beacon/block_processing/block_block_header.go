package block_processing

import (
	"errors"
	"github.com/protolambda/zrnt/eth2/beacon"
	"github.com/protolambda/zrnt/eth2/util/bls"
	"github.com/protolambda/zrnt/eth2/util/ssz"
)

func ProcessBlockHeader(state *beacon.BeaconState, block *beacon.BeaconBlock) error {
	// Verify that the slots match
	if block.Slot != state.Slot {
		return errors.New("slot of block does not match slot of state")
	}
	// Verify that the parent matches
	prevRoot := ssz.HashTreeRoot(state.LatestBlockHeader)
	if block.PreviousBlockRoot != prevRoot {
		return errors.New("previous block root does not match root from latest state block header")
	}
	// Save current block as the new latest block
	state.LatestBlockHeader = block.GetTemporaryBlockHeader()

	propIndex := state.GetBeaconProposerIndex(state.Slot, false)
	// Verify proposer signature
	proposer := &state.ValidatorRegistry[propIndex]
	// Block signature
	if !bls.BlsVerify(
		proposer.Pubkey,
		ssz.SignedRoot(block),
		block.Signature,
		beacon.GetDomain(state.Fork, state.Epoch(), beacon.DOMAIN_BEACON_BLOCK)) {
		return errors.New("block signature invalid")
	}
	return nil
}
