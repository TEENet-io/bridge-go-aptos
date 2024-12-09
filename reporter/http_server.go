// This is a http type of reporter.
// It fetches data from internal state/statedb
// and publishes on the http routes.

package reporter

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/common"
	"github.com/TEENet-io/bridge-go/state"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

const (
	ROUTE_HELLO   = "/hello"
	ROUTE_DEPOSIT = "/deposit"
)

type HttpReporter struct {
	serverIP   string // listen ip
	serverPort string // listen port

	// upstream data sources
	depositdb btcaction.DepositStorage // this is an interface
	statedb   *state.StateDB
}

func NewHttpReporter(serverIP string, serverPort string, depositdb btcaction.DepositStorage, statedb *state.StateDB) *HttpReporter {
	return &HttpReporter{
		serverIP:   serverIP,
		serverPort: serverPort,
		depositdb:  depositdb,
		statedb:    statedb,
	}
}

// Hook up routes & handlers
func (h *HttpReporter) SetupRouter() *gin.Engine {
	router := gin.Default()

	// Define routes & handlers
	router.GET(ROUTE_HELLO, Hello)
	router.GET(ROUTE_DEPOSIT, h.Deposit)

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

// func main() {
//     // Example usage
//     depositdb := &btcaction.DepositStorage{}
//     statedb := &state.StateDB{}
//     reporter := NewHttpReporter("0.0.0.0", "8080", depositdb, statedb)
//     reporter.Run()
// }
