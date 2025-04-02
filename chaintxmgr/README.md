TxMgr shall do following jobs:

### 1) Maintain `MgrState` database (not `State` database)

The state tracks txs and their post-status, whether txs is mined by the blockchain or not. The reason for their rejection, etc. This database shall be as "common" as possible since different blockchains are using this same database.

### 2) Mint

- Read `state`, find new mints; (`.GetUnminted()`)
- Compare with the records in `MgrState`, filter out already-sent mints.
- Send mints; (this step uses etherman/aptosman)
- Update `MgrState`, put newly sent mint as `monitoring` status. (no update of `state`)

### 3) RedeemPrepare (Prepare tx, 2nd-half of real redeem)

- Read `state`, find redeems that are requested but not redeemed. (`.GetRedeemsByStatus(status)`)
- Compare with records in `MgrState`, filter out already-sent redeemPrepares.
- Send redeemPrepare. (this step uses etherman/aptosman)
- Update `MgrState`, put newly sent mint as `monitoring` status. (no update of `state`)

### 4) Monitor Tx status

See if "pending" tx is "success", or "rejected", or tx has timed out etc.