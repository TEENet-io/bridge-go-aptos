### priv keys -> sim_ether_man

1) contains: a full simulated eth chain
2) contains: eth accounts with some money
3) contains: a random btc private key - pub key (For schnorr)
4) did: deployed smart contracts to the internal simulated eth chain.
5) contains: a useful sub-object of "Etherman" type. this operates on the eth chain.

## eth side, core components

### statedb (pure db)

-> sqldb -> file

### state

-> statedb
-> evm_chain_id

### eth_tx_manager_db (pure db)

-> sqldb

### eth_synchronizer (read info from evm chain, change state)

-> Etherman (a type of)
-> state
-> evm_chain_id

functional details:
- use "etherman" to scan for 1) finalized blocks, 2) minted, requested, prepared event logs.
- trigger "state" for each type of event log.

### eth_tx_manager (operates on evm chain, send tx)

-> Etherman (a type of)
-> statedb
-> eth_tx_manager_db
-> Schnorr wallet (?)
-> BTC wallet (?)

functional details:
- use "etherman" to find out "isMinted", "IsPrepared". Do dirty jobs like "Mint", "RedeemPrepare".
- use "btcwallet" to query and lock some BTC (redeem process: prepare stage).
- use "eth_tx_manager_db" to track monitored tx.
- use "statedb" to find "not minted yet" db records, "user requested but not prepared" db records.
- use "schnorrwallet" to sign Mint Tx & Prepare Tx.


So techniqually:
1) the deployment of smart contracts is BEFORE the eth side core components are running.
2) the money and accounts are already there.
3) the etherman (regardless of simulated or not), shall be able to deal with the evm chain.

Rewrite:

simEtherman -> realEtherman