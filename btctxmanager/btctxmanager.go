package btctxmanager

import (
	"strings"
	"time"

	"github.com/TEENet-io/bridge-go/btcman/assembler"
	"github.com/TEENet-io/bridge-go/btcman/rpc"
	"github.com/TEENet-io/bridge-go/btcman/utxo"
	"github.com/TEENet-io/bridge-go/btcvault"
	"github.com/TEENet-io/bridge-go/state"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

const (
	QUERY_DB_INTERVAL = 10 * time.Second
	BTC_TX_FEE        = int64(0.001 * 1e8)
)

// Remove0xPrefix removes the "0x" prefix from a string if it exists
// It is helpful because BTC address/txid don't have the "0x" prefix.
func Remove0xPrefix(input string) string {
	if strings.HasPrefix(input, "0x") {
		return input[2:]
	}
	return input
}

type BtcTxManager struct {
	treasureVault btcvault.TreasureVault // where to query details of UTXOs.
	legacySigner  assembler.LegacySigner // who to sign the txs.
	myBtcClient   rpc.RpcClient          // send/query btc blockchain.
	sharedState   state.State            // fetch and update the shared state.
}

// Find "prepared" redeems from local shared "state"
func (m *BtcTxManager) FindRedeems() ([]*state.Redeem, error) {
	redeems, err := m.sharedState.GetPreparedRedeems()
	if err != nil {
		return nil, err
	}
	return redeems, nil
}

// FetchUTXOs fetches details of each UTXO for a single given redeem.
// It won't query the Bitcoin PRC, just fetch them from the local utxo vault.
func (m *BtcTxManager) FetchUTXOs(redeem *state.Redeem) ([]*btcvault.VaultUTXO, error) {
	var utxos []*btcvault.VaultUTXO
	for _, outpoint := range redeem.Outpoints {
		clean_txid := outpoint.TxId.String()
		utxo, err := m.treasureVault.GetUTXODetail(clean_txid, int32(outpoint.Idx))
		if err != nil {
			return nil, err
		}
		utxos = append(utxos, utxo)
	}
	return utxos, nil
}

// ConvertUTXO converts a btcvault.VaultUTXO to a utxo.UTXO
// The wallet depends on utxo.UTXO type to sign transactions.
func ConvertUTXO(vaultUTXO *btcvault.VaultUTXO) *utxo.UTXO {
	txHash, _ := chainhash.NewHashFromStr(vaultUTXO.TxID)
	return &utxo.UTXO{
		TxID:      vaultUTXO.TxID,
		TxHash:    txHash,
		Vout:      uint32(vaultUTXO.Vout),
		Amount:    vaultUTXO.Amount,
		PkScriptT: utxo.ANY_SCRIPT_T,
		PkScript:  vaultUTXO.PkScript,
	}
}

// CollectUTXOs fetches UTXOs for a given redeem and converts them to utxo.UTXO type.
func (m *BtcTxManager) CollectUTXOs(redeem *state.Redeem) ([]*utxo.UTXO, error) {
	vaultUTXOs, err := m.FetchUTXOs(redeem)
	if err != nil {
		return nil, err
	}

	var utxos []*utxo.UTXO
	for _, vaultUTXO := range vaultUTXOs {
		utxos = append(utxos, ConvertUTXO(vaultUTXO))
	}

	return utxos, nil
}

// CreateBTCRedeem creates a redeem transaction for the given redeem.
func (m *BtcTxManager) CreateBTCRedeem(redeem *state.Redeem) (*wire.MsgTx, error) {
	// Collect UTXOs to be spent
	utxos, err := m.CollectUTXOs(redeem)
	if err != nil {
		return nil, err
	}

	var requestTxHash [32]byte
	copy(requestTxHash[:], redeem.RequestTxHash.Bytes())

	redeemTx, err := m.legacySigner.MakeRedeemTx(
		Remove0xPrefix(redeem.Receiver),
		redeem.Amount.Int64(),
		requestTxHash, // we just fill in the eth redeem request tx hash as the identifier.
		m.legacySigner.P2PKH.EncodeAddress(),
		BTC_TX_FEE, // TODO: remove hard code of fee.
		utxos,
	)
	if err != nil {
		return nil, err
	}

	return redeemTx, nil
}

// WithdrawBTC sends a redeem transaction to the Bitcoin network.
func (m *BtcTxManager) WithdrawBTC(redeem *state.Redeem) (*chainhash.Hash, error) {

	redeemTx, err := m.CreateBTCRedeem(redeem)
	if err != nil {
		return nil, err
	}

	txHash, err := m.myBtcClient.SendRawTx(redeemTx)
	if err != nil {
		return nil, err
	}

	return txHash, nil
}
