Prototype

Components

- [x] State (shared between BTC and ETH)
- [x] ETH Synchronizer
- [x] ETH Tx Manager (Mint, Redeem)
- [x] BTC Vault (track spendable BTC UTXO, replace btcwallet)
- [x] BTC Tx Manager (Redeem)
- [x] BTC Synchronizer

Test Components

- [x] end-to-end test.go
- [x] BTC Deposit Sender.go
- [x] BTC reg-test Node
- [x] TWBTC Redeem Sender.go
- [x] ETH simulate Node (use memory chain)
- [x] Logger

Live Components

- [x] BTC Testnet (connect to BTC Testnet)
- [x] ETH Testnet (connect to ETH Testnet: sepolia)
- [x] server.go
- [x] btc_user.go
- [x] eth_user.go

Test Accounts

Ethereum (address+private key)

```
Bridge's ETH Wallet.
0x85b427C84731bC077BA5A365771D2b64c5250Ac8
dbcec79f3490a6d5d162ca2064661b85c40c93672968bfbd906b952e38c3e8de

User's ETH Wallet
0xdab133353Cff0773BAcb51d46195f01bD3D03940
e751da9079ca6b4e40e03322b32180e661f1f586ca1914391c56d665ffc8ec74

0xf54340017f8449Ffe11594144B1c5947D84A4323
620ec29109722a03d3c11a62dbba153ebb49716c9ec5301dae7de35648542da4

0x5E3906EFaff0a1c098a0354AfAcae508f18Cc134
90e80ef178da14140f1b15df1fb404de5cc3c35859d43dd9e5ddf472e2fa09c5

0xB6AB5442EC8E812Adee4e1EA51313a54c5064E2A
bcb90227d0058d7f72867d43b67f7196c91c06b761dad7dde379223c5409b0a5
```

Bitcoin Regtest Accounts (address+private key)


```c
// Act as the coinbase receiver (block mines and reward goes to this address)
mkVXZnqaaKt4puQNr4ovPHYg48mjguFCnT
cNSHjGk52rQ6iya8jdNT9VJ8dvvQ8kPAq5pcFHsYBYdDqahWuneH

// user's btc wallet
moHYHpgk4YgTCeLBmDE2teQ3qVLUtM95Fn
cQthTMaKUU9f6br1hMXdGFXHwGaAfFFerNkn632BpGE6KXhTMmGY

// bridge's btc wallet
mvqq54khZQta7zDqFGoyN7BVK7Li4Xwnih
cUWcwxzt2LiTxQCkQ8FKw67gd2NuuZ182LpX9uazB93JLZmwakBP

// P2TR address (dumped from bitcoin-core software)
bcrt1qvnm6etkkgmyj425hmtdmu4h82zpxudvya4y24n
cUcHsdBfXphhqLayGuxULxJeABDX74kMtL2gdfyUMVeke3ZJsKQ6
```

Bicoin Testnet4 Accounts

```c
// user's btc wallet
mtcpH6hvvCjtKz6zYFZm5Cbqg1kgi5a9tf
cUvcSxBzLSikMmewez4dY1Ucxyste3Cf2ZJsqzD5RxmVPWF8r4MV

// bridge's btc wallet
mnQ9tBEkNXXEyJqKeSK1TWJV3LngVSjanV
cU78RfXmYEXsdNpiC8AppdpNg6Ni58s8nF8LFFWuMVAQGx51v3HY

// other #1
mpuvVJxfbnwYej7NmAwEb1wFhMiq6VKzNH
cRxEqXANSYzm3aFQigEnXxWTdde3Kwcuy2vX4n1hETDnFKZpAvTQ

// other #2
mk36ppADLBQFztXbsW2WkFMGCTK2hgJXge
cVCWa2dzvhVrw2GEih4zSXdxBTmrNghXKFuTHnP5WaiK8zDeTGmF

```

## Problems:

- [ ] BTC lastblock height is not fetched in db, but using the "latest" height on the BTC Chain it is viewing.
- [ ] BTC components doesn't use ctx as stop signal, better use for graceful shutdown.
- [ ] Unused code in project. Use tool (staticcheck or golangci-lint) to find and remove them, or remove them manually.
- [ ] Eth side `Id` field of type `MonitoredTx` is used of different purposes, shall separate. not REUSED.
- [x] `ethtxmanager.MonitoredTx`:`sentAfter` breaks the monitor logic. <Level-0> bug. The last block hash is stored. However, in real life (not sim), the blck hash is not search-able. So better using last block height (int64) as a back up of hash. This now breaks the logic of finding expired Txs in local geth (regtest mode), but doesn't affect Sepolia Testnet or SimEtherman.
- [x] Automatic `ImportPrivateKeyRescan` and `ImportAddressRescan` on BTC node (regardless of local private node or regtest node) to tell BTC node wallet to track on specific address. Otherwise the BTC RPC node will not track the address (so our rpc balance / utxo query will return empty).
- [x] Need more config fileds in YAML of on server config to prevent new-deploy of smart contracts, use the existing smart contracts.
- [ ] Move Btc Regtest mining function to automatic step, no need for users to mine manually.
- [x] View Balance of btc/eth user shall contain address.
- [ ] Remove hard coded redeem fee: `BTC_TX_FEE` (btctxmanager/withdraw.go) = 0.001, `SAFE_MARGIN` (btcvault/vault.go) = 0.001.
- [ ] ETH side: synchronizer shall start not from 0 but from  a predefined block height, if not the last block height (if db is empty)
- [ ] ETH side: "state missing" latest chain id (1337) + latest block (start from 0), shall we start from a specific number, to avoid full-scan of blockchain (like after the blk of deployment block of bridge/twbtc smart contract)?
- [ ] Estimate BTC Tx fee (vbyte) based on the network! 1000 sat = 3.79v (in deposit)
- [ ] Make btc_finalized_number / BTC_MATURE_OFFSET a configurable int. (now uses 1 or 0)
- [x] ETH sync: shows prematurely "stopping Eth synchronization" in sepolia environment. must have some problems. - problem was rpc node doesn't support method "eth_getlogs" and existed prematurely. So we use a more robust node instead.
- [x] BTC new config: forceStartBlk, trigger birige scan from this blk, not newest, nor from 0 blk.
- [ ] BTC Deposit Action didn't log the sender. (probably log the Tx sender)
```
Bitcoin transactions do not include a dedicated "sender" field. Instead, every input in a transaction spends an output from a previous transaction. This means:

- There is no single sender address; if multiple inputs are used, there could be multiple originating addresses.
- To infer the sender(s), you must look up each input's referenced previous output and then derive the address from the locking script.
- For common types (e.g. P2PKH), you can often extract the public key hash from the input’s signature script, but this involves parsing the script and is not as straightforward as an explicit field.

So, in your loop through block.Transactions, you'll need to inspect each transaction's inputs and perform additional lookups or script parsing if you want to determine the originating addresses.
```
