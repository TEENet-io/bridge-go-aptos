package state

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	logger "github.com/sirupsen/logrus"
)

// Fetch ETH Finalized Block from the state db.
// If not found, use 0 instead.
// Set the value to st.cache
func (st *State) initEthFinalizedBlock() error {
	storedBytes32, ok, err := st.statedb.GetKeyedValue(KeyEthFinalizedBlock)
	if err != nil {
		return ErrGetEthFinalizedBlockNumber
	}

	if !ok {
		logger.WithField("default", common.EthStartingBlock).Warn("State: Missing evm lastest block #")
		// save the default value
		err := st.statedb.SetKeyedValue(KeyEthFinalizedBlock, common.BigInt2Bytes32(common.EthStartingBlock))
		if err != nil {
			return ErrSetEthFinalizedBlockNumber
		}
		st.cache.lastEthFinalized.Store(common.EthStartingBlock.Bytes())
	} else {
		stored := new(big.Int).SetBytes(storedBytes32[:])
		logger.WithField("evm_last_block", stored.Int64()).Info("State: Loaded evm last block from db #")
		// stored value must not be less than the starting block number
		if stored.Cmp(common.EthStartingBlock) == -1 {
			logger.Errorf("stored last finalized block number is invalid: %v", stored)
			return ErrStoredEthFinalizedBlockNumberInvalid
		}
		st.cache.lastEthFinalized.Store(stored.Bytes())
	}
	return nil
}

// Same as above.
// Fetch ETH chain ID, if not found set the default value.
func (st *State) initEthChainID() error {
	storedBytes32, ok, err := st.statedb.GetKeyedValue(KeyEthChainId)
	if err != nil {
		return ErrGetEthChainId
	}

	if !ok {
		logger.WithField("chainId", st.cfg.UniqueChainId).Warn("state: Missing evm chainId, use default")
		// save the default value
		err := st.statedb.SetKeyedValue(KeyEthChainId, common.BigInt2Bytes32(st.cfg.UniqueChainId))
		if err != nil {
			return ErrSetEthChainId
		}
		st.cache.ethChainId.Store(st.cfg.UniqueChainId.Bytes())
	} else {
		stored := new(big.Int).SetBytes(storedBytes32[:])
		logger.WithField("evm_chainId", stored.Int64()).Info("State: Load evm chainId from db #")

		if stored.Cmp(st.cfg.UniqueChainId) != 0 {
			logger.Errorf("current chain id does not match the stored: curr=%v, stored=%v", st.cfg.UniqueChainId, stored)
			return ErrEthChainIdUnmatchedStored
		}
		st.cache.ethChainId.Store(stored.Bytes())
	}
	return nil
}
