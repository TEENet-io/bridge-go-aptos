// This is a http type of reporter.
// It fetches data from internal state/statedb
// and publishes on the http routes.

package reporter

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	logger "github.com/sirupsen/logrus"

	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/btcman/utils"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

const (
	ROUTE_HELLO    = "/hello"
	ROUTE_DEPOSITS = "/deposits"
	ROUTE_REDEEMS  = "/redeems"
)

type HttpReporter struct {
	serverIP   string // listen ip
	serverPort string // listen port

	// upstream data sources

	// BTC side.
	depositdb btcaction.DepositStorage      // this is an interface
	redeemdb  btcaction.RedeemActionStorage // this is an interface

	// ETH side.
	statedb *state.StateDB
}

func NewHttpReporter(serverIP string, serverPort string, depositdb btcaction.DepositStorage, redeemdb btcaction.RedeemActionStorage, statedb *state.StateDB) *HttpReporter {
	return &HttpReporter{
		serverIP:   serverIP,
		serverPort: serverPort,
		depositdb:  depositdb,
		redeemdb:   redeemdb,
		statedb:    statedb,
	}
}

// Hook up routes & handlers
func (h *HttpReporter) SetupRouter() *gin.Engine {
	router := gin.Default()

	// Define routes & handlers
	router.GET(ROUTE_HELLO, Hello)
	router.GET(ROUTE_DEPOSITS, h.Deposits)
	router.GET(ROUTE_REDEEMS, h.Redeems)

	return router
}

// Hook up router & ip:port
func (h *HttpReporter) Run() {
	router := h.SetupRouter()
	address := h.serverIP + ":" + h.serverPort
	if err := router.Run(address); err != nil {
		panic(err)
	}
}

// Example route.
func Hello(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "world",
	})
}

type DepositResponse struct {
	BtcDepoTxStatus string `json:"btc_depo_tx_status"` // "not_found", "pending", "confirmed"
	BtcDepoTxId     string `json:"btc_depo_tx_id"`     // btc transaction id
	BtcDepoAmount   string `json:"btc_depo_amount"`    // btc deposit amount in Satoshi, int64 => string

	EvmMintTxStatus string `json:"evm_mint_tx_status"` // "not_found", "pending", "confirmed"
	EvmMintReceiver string `json:"evm_mint_receiver"`  // ethereum address
	EvmMintTxId     string `json:"evm_mint_tx_id"`     // ethereum mint transaction id
	EvmMintAmount   string `json:"evm_mint_amount"`    // ethereum mint amount in Wei, int64 => string
}

// Fetch a list of deposits.
// deposits are related to an account, and ordered by time.
// evm_receiver: ethereum address of receiver on eth side
func (h *HttpReporter) Deposits(c *gin.Context) {
	evmReceiver := c.Query("evm_receiver") // evm

	// Check if all parameters are missing
	if evmReceiver == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "evm_receiver must be provided"})
		return
	}

	// Protect / Shape the input
	evmReceiver = strings.ToLower(evmReceiver)
	// Acutally deposits has 0x as prefix when storing in the db.
	// evmReceiver = utils.Remove0xPrefix(evmReceiver)

	var resp []DepositResponse

	// Query the depositdb via evmReceiver
	depos, err := h.depositdb.GetDepositsByEVMAddr(evmReceiver)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.WithField("len(depos)", len(depos)).Info("Query Depos from db")
	if len(depos) == 0 {
		// order depos by blocknumber desc
		sort.Slice(depos, func(i, j int) bool {
			return depos[i].BlockNumber > depos[j].BlockNumber
		})
	}

	for _, depo := range depos {
		_resp := DepositResponse{
			BtcDepoTxStatus: "confirmed",
			BtcDepoTxId:     depo.TxHash,
			BtcDepoAmount:   strconv.FormatInt(depo.DepositValue, 10), // convert int64 to string
		}

		mint, _, err := h.statedb.GetMint(ethcommon.HexToHash(depo.TxHash))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// If no corresponding mint on evm was found.
		// We don't fill in the evm part of response
		if mint == nil {
			continue
		}

		// found mint, but the evm tx hash is not set.
		if mint.MintTxHash == common.EmptyHash {
			_resp.EvmMintTxStatus = "pending"
			_resp.EvmMintReceiver = depo.EvmAddr
		} else { // found mint, evm tx hash is set.
			_resp.EvmMintTxStatus = "confirmed"
			_resp.EvmMintReceiver = depo.EvmAddr
			_resp.EvmMintTxId = mint.MintTxHash.String()
			_resp.EvmMintAmount = mint.Amount.String()
		}

		// attach _resp to response
		resp = append(resp, _resp)
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

type RedeemResponse struct {
	EvmRequester     string `json:"evm_requester"`      // evm requester
	EvmRequestTxId   string `json:"evm_request_tx_id"`  // evm request transaction id
	EvmRequestAmount string `json:"evm_request_amount"` // evm request amount in Wei, int64 => string

	EvmPrepareTxId string `json:"evm_prepare_tx_id"` // evm prepare transaction id

	BtcRedeemReceiver string `json:"btc_redeem_receiver"` // btc receiver address
	BtcRedeemTxId     string `json:"btc_redeem_tx_id"`    // btc redeem transaction id
	BtcRedeemAmount   string `json:"btc_redeem_amount"`   // btc redeem amount in Satoshi, int64 => string
	BtcRedeemStatus   string `json:"btc_redeem_status"`   // btc redeem status, one of "sent", "mined"

	Status string `json:"status"` // overall status, one of "requested/prepared/completed/invalid"
}

// Fetch data from redeemdb + statedb,
// assemble response and return.
func (h *HttpReporter) Redeems(c *gin.Context) {
	evmRequester := c.Query("evm_requester") // evm requester

	if evmRequester == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "evm_requester must be provided"})
		return
	}

	if !common.EnsureSafeAddressHexString(evmRequester) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "evm_requester is not a valid address string"})
	}

	logger.WithField("evmRequester", evmRequester).Info("Redeem Route")

	// Fetch redeems by evm requester
	redeems, err := h.statedb.GetRedeemsByRequester(common.Trim0xPrefix(evmRequester))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.WithField("len(redeems)", len(redeems)).Info("Redeems Route")

	var response []RedeemResponse
	for _, redeem := range redeems {
		// phase one: requested
		_resp := RedeemResponse{
			EvmRequester:     common.Prepend0xPrefix(common.ByteSliceToPureHexStr(redeem.Requester)),
			EvmRequestTxId:   redeem.RequestTxHash.String(),
			EvmRequestAmount: redeem.Amount.Text(10),
			Status:           "requested",
		}
		// phase two: prepared
		if redeem.PrepareTxHash != common.EmptyHash {
			_resp.EvmPrepareTxId = redeem.PrepareTxHash.String()
			_resp.Status = "prepared"
		}

		// phase three: unsent, send, mined
		if redeem.Receiver != "" {
			_resp.BtcRedeemReceiver = redeem.Receiver
			_resp.BtcRedeemAmount = redeem.Amount.Text(10)
		}

		// If redeem is executed & found on BTC side.
		_requestTxHash := utils.Remove0xPrefix(redeem.RequestTxHash.String())
		hasIt, err := h.redeemdb.HasRedeem(_requestTxHash)

		logger.WithField("hasIt", hasIt).Debug("Redeem Route")

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if !hasIt { // BTC side hasn't prepare or execute the redeem.
			response = append(response, _resp)
			continue // shortcut
		}

		_redeemAction, err := h.redeemdb.QueryByEthRequestTxId(_requestTxHash)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_resp.BtcRedeemTxId = _redeemAction.BtcHash

		if _redeemAction.Sent {
			_resp.BtcRedeemStatus = "sent"
		}
		if _redeemAction.Mined {
			_resp.BtcRedeemStatus = "mined"
			_resp.Status = "completed"
		}

		response = append(response, _resp)
	}

	c.JSON(http.StatusOK, gin.H{"data": response})
}

// func main() {
//     // Example usage
//     depositdb := &btcaction.DepositStorage{}
//     statedb := &state.StateDB{}
//     reporter := NewHttpReporter("0.0.0.0", "8080", depositdb, statedb)
//     reporter.Run()
// }
