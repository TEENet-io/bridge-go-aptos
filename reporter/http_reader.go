// Reader is a testing facility to read the output of a http reporter.

package reporter

import (
	"io"
	"net/http"
)

type HttpReader struct {
	serverIP   string // listen ip
	serverPort string // listen port
}

func NewHttpReader(serverIP string, serverPort string) *HttpReader {
	return &HttpReader{
		serverIP:   serverIP,
		serverPort: serverPort,
	}
}

func (hr *HttpReader) GetHello() (string, error) {
	url := "http://" + hr.serverIP + ":" + hr.serverPort + ROUTE_HELLO

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Convert the body to a string
	return string(body), nil
}

func (hr *HttpReader) GetDepositStatus(btcTxID string) (string, error) {
	url := "http://" + hr.serverIP + ":" + hr.serverPort + ROUTE_DEPOSIT + "?btc_tx_id=" + btcTxID
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Convert the body to a string
	return string(body), nil
}
