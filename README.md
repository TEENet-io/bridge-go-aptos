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

- [ ] BTC Testnet (connect to BTC Testnet)
- [ ] ETH Hardhat / Testnet (connect to ETH Testnet)
- [ ] server.go
- [ ] btc_user.go
- [ ] eth_user.go

Test Accounts

Ethereum (address+private key)

```
0x85b427C84731bC077BA5A365771D2b64c5250Ac8
dbcec79f3490a6d5d162ca2064661b85c40c93672968bfbd906b952e38c3e8de

0xdab133353Cff0773BAcb51d46195f01bD3D03940
e751da9079ca6b4e40e03322b32180e661f1f586ca1914391c56d665ffc8ec74

0xf54340017f8449Ffe11594144B1c5947D84A4323
620ec29109722a03d3c11a62dbba153ebb49716c9ec5301dae7de35648542da4

0x5E3906EFaff0a1c098a0354AfAcae508f18Cc134
90e80ef178da14140f1b15df1fb404de5cc3c35859d43dd9e5ddf472e2fa09c5

0xB6AB5442EC8E812Adee4e1EA51313a54c5064E2A
bcb90227d0058d7f72867d43b67f7196c91c06b761dad7dde379223c5409b0a5
```

Bitcoin Regtest (address+private key)


```c
// This btc wallet holds a lot of money.
// Also acts the coinbase receiver (block mines and reward goes to this address)
mkVXZnqaaKt4puQNr4ovPHYg48mjguFCnT
cNSHjGk52rQ6iya8jdNT9VJ8dvvQ8kPAq5pcFHsYBYdDqahWuneH

// user's btc wallet
moHYHpgk4YgTCeLBmDE2teQ3qVLUtM95Fn
cQthTMaKUU9f6br1hMXdGFXHwGaAfFFerNkn632BpGE6KXhTMmGY


// bridge's btc wallet
mvqq54khZQta7zDqFGoyN7BVK7Li4Xwnih
cUWcwxzt2LiTxQCkQ8FKw67gd2NuuZ182LpX9uazB93JLZmwakBP
```