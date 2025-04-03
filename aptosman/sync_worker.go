package aptosman

import (
	"math/big"

	"github.com/TEENet-io/bridge-go/agreement"
	"github.com/TEENet-io/bridge-go/common"
)

// AptosSyncWorker 实现 chainsync.SyncWorker 接口
type AptosSyncWorker struct {
	aptosman *Aptosman
}

// NewAptosSyncWorker 创建新的 Aptos 同步工作器
func NewAptosSyncWorker(aptosman *Aptosman) *AptosSyncWorker {
	return &AptosSyncWorker{
		aptosman: aptosman,
	}
}

// GetNewestLedgerFinalizedNumber 实现接口方法，获取最新的已确认版本号
func (w *AptosSyncWorker) GetNewestLedgerFinalizedNumber() (*big.Int, error) {
	version, err := w.aptosman.GetLatestFinalizedVersion()
	if err != nil {
		return nil, err
	}
	return big.NewInt(int64(version)), nil
}

// GetTimeOrderedEvents 实现接口方法，获取按时间顺序排列的事件
func (w *AptosSyncWorker) GetTimeOrderedEvents(oldNum *big.Int, newNum *big.Int) (
	[]agreement.MintedEvent,
	[]agreement.RedeemRequestedEvent,
	[]agreement.RedeemPreparedEvent,
	error,
) {
	// 获取事件
	mintedEvents, requestedEvents, preparedEvents, err := w.aptosman.GetModuleEvents(
		oldNum.Uint64(),
		newNum.Uint64(),
	)
	if err != nil {
		return nil, nil, nil, err
	}

	// 转换 MintedEvent
	agreementMinted := make([]agreement.MintedEvent, 0, len(mintedEvents))
	for _, ev := range mintedEvents {
		agreementMinted = append(agreementMinted, agreement.MintedEvent{
			MintTxHash: common.HexStrToBytes32(ev.MintTxHash),
			BtcTxId:    common.HexStrToBytes32(ev.BtcTxId),
			Amount:     new(big.Int).SetUint64(ev.Amount),
			Receiver:   []byte(ev.Receiver),
		})
	}

	// 转换 RedeemRequestedEvent
	agreementRequested := make([]agreement.RedeemRequestedEvent, 0, len(requestedEvents))
	for _, ev := range requestedEvents {
		agreementRequested = append(agreementRequested, agreement.RedeemRequestedEvent{
			RequestTxHash:   common.HexStrToBytes32(ev.RequestTxHash),
			Requester:       []byte(ev.Requester),
			Amount:          new(big.Int).SetUint64(ev.Amount),
			Receiver:        ev.Receiver,
			IsValidReceiver: common.IsValidBtcAddress(ev.Receiver, w.aptosman.cfg.BtcChainConfig),
		})
	}

	// 转换 RedeemPreparedEvent
	agreementPrepared := make([]agreement.RedeemPreparedEvent, 0, len(preparedEvents))
	for _, ev := range preparedEvents {
		outpointTxIds := make([][32]byte, len(ev.OutpointTxIds))
		for i, txId := range ev.OutpointTxIds {
			outpointTxIds[i] = common.HexStrToBytes32(txId)
		}

		agreementPrepared = append(agreementPrepared, agreement.RedeemPreparedEvent{
			PrepareTxHash: common.HexStrToBytes32(ev.PrepareTxHash),
			RequestTxHash: common.HexStrToBytes32(ev.RequestTxHash),
			Requester:     []byte(ev.Requester),
			Receiver:      ev.Receiver,
			Amount:        new(big.Int).SetUint64(ev.Amount),
			OutpointTxIds: common.ArrayHexStrToHashes(ev.OutpointTxIds),
			OutpointIdxs:  ev.OutpointIdxs,
		})
	}

	return agreementMinted, agreementRequested, agreementPrepared, nil
}
