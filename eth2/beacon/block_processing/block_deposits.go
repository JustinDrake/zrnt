package block_processing

import (
	"errors"
	"fmt"
	"github.com/protolambda/zrnt/eth2/beacon"
	"github.com/protolambda/zrnt/eth2/util/bls"
	"github.com/protolambda/zrnt/eth2/util/hash"
	"github.com/protolambda/zrnt/eth2/util/merkle"
	"github.com/protolambda/zrnt/eth2/util/ssz"
)

func ProcessBlockDeposits(state *beacon.BeaconState, block *beacon.BeaconBlock) error {
	if len(block.Body.Deposits) > beacon.MAX_DEPOSITS {
		return errors.New("too many deposits")
	}
	for _, dep := range block.Body.Deposits {
		if err := ProcessDeposit(state, &dep); err != nil {
			return err
		}
		state.DepositIndex += 1
	}
	return nil
}

// Process a deposit from Ethereum 1.0.
func ProcessDeposit(state *beacon.BeaconState, dep *beacon.Deposit) error {
	depositInput := &dep.DepositData.DepositInput

	// Deposits must be processed in order
	if dep.Index != state.DepositIndex {
		return errors.New(fmt.Sprintf("deposit has index %d that does not match with state index %d", dep.Index, state.DepositIndex))
	}

	serializedDepositData := dep.DepositData.Serialized()

	// Verify the Merkle branch
	if !merkle.VerifyMerkleBranch(
		hash.Hash(serializedDepositData),
		dep.Proof[:],
		beacon.DEPOSIT_CONTRACT_TREE_DEPTH,
		uint64(dep.Index),
		state.LatestEth1Data.DepositRoot) {
		return errors.New(fmt.Sprintf("deposit %d has merkle proof that failed to be verified", dep.Index))
	}

	// Increment the next deposit index we are expecting. Note that this
	// needs to be done here because while the deposit contract will never
	// create an invalid Merkle branch, it may admit an invalid deposit
	// object, and we need to be able to skip over it
	state.DepositIndex += 1

	if !bls.BlsVerify(
		depositInput.Pubkey,
		ssz.SignedRoot(depositInput),
		depositInput.ProofOfPossession,
		beacon.GetDomain(state.Fork, state.Epoch(), beacon.DOMAIN_DEPOSIT)) {
		// simply don't handle the deposit. (TODO: should this be an error (making block invalid)?)
		return nil
	}

	valIndex := beacon.ValidatorIndexMarker
	for i, v := range state.ValidatorRegistry {
		if v.Pubkey == depositInput.Pubkey {
			valIndex = beacon.ValidatorIndex(i)
			break
		}
	}

	amount := dep.DepositData.Amount
	withdrawalCredentials := depositInput.WithdrawalCredentials
	// Check if it is a known validator that is depositing ("if pubkey not in validator_pubkeys")
	if valIndex == beacon.ValidatorIndexMarker {
		// Not a known pubkey, add new validator
		validator := beacon.Validator{
			Pubkey:                depositInput.Pubkey,
			WithdrawalCredentials: withdrawalCredentials,
			ActivationEpoch:       beacon.FAR_FUTURE_EPOCH, ExitEpoch: beacon.FAR_FUTURE_EPOCH, WithdrawableEpoch: beacon.FAR_FUTURE_EPOCH,
			InitiatedExit: false, Slashed: false,
		}
		// Note: In phase 2 registry indices that have been withdrawn for a long time will be recycled.
		state.ValidatorRegistry = append(state.ValidatorRegistry, validator)
		state.ValidatorBalances = append(state.ValidatorBalances, amount)
	} else {
		// known pubkey, check withdrawal credentials first, then increase balance.
		if state.ValidatorRegistry[valIndex].WithdrawalCredentials != withdrawalCredentials {
			return errors.New("deposit has wrong withdrawal credentials")
		}
		// Increase balance by deposit amount
		state.ValidatorBalances[valIndex] += amount
	}
	return nil
}
