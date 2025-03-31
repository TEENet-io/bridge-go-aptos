package state

import (
	"database/sql"
	"math/big"

	"github.com/TEENet-io/bridge-go/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	logger "github.com/sirupsen/logrus"
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

func RandMint(isMinted bool) *Mint {
	var txHash ethcommon.Hash
	if isMinted {
		txHash = common.RandBytes32()
	}

	return &Mint{
		BtcTxId:    common.RandBytes32(),
		MintTxHash: txHash,
		Receiver:   common.RandEthAddress().Bytes(),
		Amount:     big.NewInt(100),
	}
}

func getMemoryDB() *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		logger.Fatal(err)
	}
	return db
}
