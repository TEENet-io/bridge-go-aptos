# Chain Tx Manager DB (chainTxMgrDB)

This db is specific used for Chain Manager module to track txs that are sent.

Whether it is Ethereum or APTOs, the chain-man sends the tx, and put the tx to here for continious monitoring.

### What Types of Txs Are Monitored?
Currently in **BTC-to-X** bridge (where X is Ethereum, Aptos, etc), the `Mint` Tx issued by bridge and `RedeemPrepare` Tx issued by bridge are tracked.

### What Aspects of Txs Are Monitored?

Whether they are "mal-formed", "rejected", "success", "timeout", "missing", etc.

### What is the Choice of underlying Database?

ChainTxMgrDB shall be supported by **ANY** database implementation. The recommended is *SQLite*. Or even a json file is suffice. There is an asbstraction layer on-top-of the real database-specific implementation. See `types.go` file.

### Files

`types.go`: define the types and interface of such db.

`sqlite_chaintxmgrdb.go`: the sqlite implementation of such db.