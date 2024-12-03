// This is a http type of reporter.
// It fetches data from internal state/statedb
// and publishes on the http routes.

package reporter

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/TEENet-io/bridge-go/btcaction"
	"github.com/TEENet-io/bridge-go/state"
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

// Fetch data from depositdb
// Publish on the route
func (h *HttpReporter) Deposit(c *gin.Context) {
	btcTxID := c.Query("btc_tx_id")
	sender := c.Query("sender")

	// Check if both parameters are missing
	if btcTxID == "" && sender == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either btc_tx_id or sender must be provided"})
		return
	}

	txs, err := h.depositdb.GetDepositByTxHash(btcTxID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(txs) > 0 {
		c.JSON(http.StatusOK, gin.H{"data": txs})
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "No deposit found"})
	}
}

// func main() {
//     // Example usage
//     depositdb := &btcaction.DepositStorage{}
//     statedb := &state.StateDB{}
//     reporter := NewHttpReporter("0.0.0.0", "8080", depositdb, statedb)
//     reporter.Run()
// }
