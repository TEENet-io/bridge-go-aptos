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
- [ ] bridge.go
- [ ] btc-user.go
- [ ] eth-user.go