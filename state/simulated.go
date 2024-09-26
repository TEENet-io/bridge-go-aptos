package state

import (
	"database/sql"
	"math/big"

	logger "github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/TEENet-io/bridge-go/common"
)

func RandRedeem(status RedeemStatus) *Redeem {
	return &Redeem{
		RequestTxHash: common.RandBytes32(),
		PrepareTxHash: common.RandBytes32(),
		BtcTxId:       common.RandBytes32(),
		Requester:     common.RandEthAddress(),
		Amount:        big.NewInt(100),
		Outpoints: []Outpoint{
			{
				TxId: common.RandBytes32(),
				Idx:  0,
			},
			{
				TxId: common.RandBytes32(),
				Idx:  1,
			},
		},
		Receiver: "rand_btc_address",
		Status:   status,
	}
}

func RandMint(status MintStatus) *Mint {
	return &Mint{
		BtcTxID:    common.RandBytes32(),
		MintTxHash: common.RandBytes32(),
		Receiver:   common.RandEthAddress(),
		Amount:     big.NewInt(100),
		Status:     status,
	}
}

func getMemoryDB() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		logger.Fatal(err)
	}
	return db
}
