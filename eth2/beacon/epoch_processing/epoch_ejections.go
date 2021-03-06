package epoch_processing

import (
	"github.com/protolambda/zrnt/eth2/beacon"
)

func ProcessEpochEjections(state *beacon.BeaconState) {
	// After we are done slashing, eject the validators that don't have enough balance left.
	for _, vIndex := range state.ValidatorRegistry.GetActiveValidatorIndices(state.Epoch()) {
		if state.ValidatorBalances[vIndex] < beacon.EJECTION_BALANCE {
			state.ExitValidator(vIndex)
		}
	}
}
