package state

import (
	"math/big"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
)

func (st *State) initEthFinalizedBlock() error {
	storedBytes32, ok, err := st.statedb.GetKeyedValue(KeyEthFinalizedBlock)
	if err != nil {
		return ErrGetEthFinalizedBlockNumber
	}

	if !ok {
		logger.Warnf("no stored last finalized block number found, using the default value %v", common.EthStartingBlock)
		// save the default value
		err := st.statedb.SetKeyedValue(KeyEthFinalizedBlock, common.BigInt2Bytes32(common.EthStartingBlock))
		if err != nil {
			return ErrSetEthFinalizedBlockNumber
		}
		st.cache.lastEthFinalized.Store(common.EthStartingBlock.Bytes())
	} else {
		stored := new(big.Int).SetBytes(storedBytes32[:])

		// stored value must not be less than the starting block number
		if stored.Cmp(common.EthStartingBlock) == -1 {
			logger.Errorf("stored last finalized block number is invalid: %v", stored)
			return ErrStoredEthFinalizedBlockNumberInvalid
		}
		st.cache.lastEthFinalized.Store(stored)
	}
	return nil
}
