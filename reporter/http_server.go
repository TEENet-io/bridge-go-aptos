// This is a http type of reporter.
// It fetches data from internal state/statedb
// and publishes on the http routes.

package reporter

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	logger "github.com/sirupsen/logrus"

	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/btcman/utils"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

const (
	ROUTE_HELLO   = "/hello"
	ROUTE_DEPOSIT = "/deposit"
	ROUTE_REDEEM  = "/redeem"
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
	router.GET(ROUTE_DEPOSIT, h.Deposit)
	router.GET(ROUTE_REDEEM, h.Redeem)

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

// Fetch data from depositdb
// Publish on the route
func (h *HttpReporter) Deposit(c *gin.Context) {
	btcTxID := c.Query("btc_tx_id")        // btc transaction id
	evmReceiver := c.Query("evm_receiver") // evm

	// Check if all parameters are missing
	if btcTxID == "" && evmReceiver == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one of btc_tx_id, or evm_receiver must be provided"})
		return
	}

	// TODO: protection of parameter check.

	// Primarily, deposits are fetched by btc tx id.
	var btcTxIDs []string

	// Assemble [btc_tx_id]
	if evmReceiver != "" {
		depos, err := h.depositdb.GetDepositByEVMAddr(evmReceiver)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, depo := range depos {
			btcTxIDs = append(btcTxIDs, depo.TxHash)
		}
	} else if btcTxID != "" {
		btcTxIDs = append(btcTxIDs, btcTxID)
	}

	// If btc_tx_id is provided, fetch a deposit by btc_tx_id.
	// will always return a single deposit response, even if not found.
	var resp []DepositResponse

	if len(btcTxIDs) != 0 {
		for _, _btcTxID := range btcTxIDs {

			depos, err := h.depositdb.GetDepositByTxHash(_btcTxID)
			// protect
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// protect
			if len(depos) > 1 {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Multiple deposits with same btcTxID found"})
				return
			}

			if len(depos) == 0 {
				resp = append(resp, DepositResponse{
					BtcDepoTxStatus: "not_found",
					BtcDepoTxId:     _btcTxID,
				})
				continue
			}

			_depo := depos[0]

			_resp := DepositResponse{
				BtcDepoTxStatus: "confirmed",
				BtcDepoTxId:     _btcTxID,
				BtcDepoAmount:   strconv.FormatInt(_depo.DepositValue, 10), // convert int64 to string
			}

			m, _, err := h.statedb.GetMint(ethcommon.HexToHash(_btcTxID))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// no corresponding mint on evm was found.
			// We don't fill in the evm part of response
			if m == nil {
				continue
			}

			// found mint, but the evm tx hash is not set.
			if m.MintTxHash == common.EmptyHash {
				_resp.EvmMintTxStatus = "pending"
				_resp.EvmMintReceiver = _depo.EvmAddr
			} else { // found mint, evm tx hash is set.
				_resp.EvmMintTxStatus = "confirmed"
				_resp.EvmMintReceiver = _depo.EvmAddr
				_resp.EvmMintTxId = m.MintTxHash.String()
				_resp.EvmMintAmount = m.Amount.String()
			}

			// attach _resp to response
			resp = append(resp, _resp)
		}

		c.JSON(http.StatusOK, gin.H{"data": resp})
	}
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
func (h *HttpReporter) Redeem(c *gin.Context) {
	evmRequester := c.Query("evm_requester") // evm requester

	if evmRequester == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "evm_requester must be provided"})
		return
	}

	logger.WithField("evmRequester", evmRequester).Info("Redeem Route")

	// Fetch redeems by evm requester
	redeems, err := h.statedb.GetRedeemsByRequester(ethcommon.HexToAddress(evmRequester))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	logger.WithField("len(redeems)", len(redeems)).Info("Redeems Route")

	var response []RedeemResponse
	for _, redeem := range redeems {
		// phase one: requested
		_resp := RedeemResponse{
			EvmRequester:     redeem.Requester.String(),
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

		logger.WithField("hasIt", hasIt).Info("Redeem Route")

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
