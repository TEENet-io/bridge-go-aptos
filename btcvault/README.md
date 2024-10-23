BTC Vault tracks the UTXOs that a specific BTC wallet can use.

# Vault

- AddUTXO() => Add UTXOs to the database.
- ChooseAndLock() => Lock up some UTXOs, prepare to be spent, at same time set timetout to a default value
- ReleaseByCommand() => Release the lock on one utxo, by specifiying txid + vout.
- ReleaseByExpire() => Scan the database and release any utxo that has expired time.
- GetUTXODetail() => Get the detail of a specific utxo.

## Interoperability with ETH Manager

Shall implement the following interface:

```go
type BtcWallet interface {
	Request(
		Id ethcommon.Hash, // eth requestTxHash
		amount *big.Int,
		ch chan<- []state.Outpoint,
	) error
}
```

## Vault: Track specs of UTXO:

- spent/unspent status (data from blockchain)
- *timeout & release (if anything goes wrong with ETH manager side)
- PkScript of UTXO. (for future spending)

## Vault: Track Overall Status

- Total sum of free & spendable UTXOs.

# ADT Design: supports various db backend

Abstract Data Type (ADT) defines the data structure, and insert/query/modify operations over table/database. On this level, the implementation is dealing with different DB backends, so the code is abstract.

see `types.go`

table vault_utxo

- block_number (int32)
- block_hash (string 64)
- tx_id (string 64),
- vout (int32),
- amount in satoshi (int64)
- lockup = True/False (bool, default False)
- spent = True/False (bool, default False)
- timeout = unix timestamp in seconds, set to 0 if untouched. after this timeout timestamp the value can be spent again.

table spent_utxo (optional)
- related_txid
- related_vout
- block_number (int32),
- block_hash (string 64),
- tx_id (string 64),
- vin (int32)

# SQLite Implementation

see `sqlite_db.go`.

Detailed implementation of above ADT in the SQLite environment.

# Vault Implemenation

Vault is the wrapper and outmost exposing entity that user should call.

see `vault.go`